package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/juliotorresmoreno/odoo-backups/config"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"vitess.io/vitess/go/vt/log"
)

type StorageClient struct {
	ClientSet *k8sclient.Clientset
	Namespace string
}

func GetClientSet() (*k8sclient.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		home := os.Getenv("HOME")
		kubeconfig := filepath.Join(home, ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}
	return k8sclient.NewForConfig(config)
}

func NewStorageClient(clientSet *k8sclient.Clientset, namespace string) *StorageClient {
	if clientSet == nil {
		var err error
		clientSet, err = GetClientSet()
		if err != nil {
			log.Fatal(fmt.Sprintf("Failed to create Kubernetes client: %v", err))
		}
	}

	return &StorageClient{
		ClientSet: clientSet,
		Namespace: namespace,
	}
}

func (s *StorageClient) ListPVCs(ctx context.Context) ([]corev1.PersistentVolumeClaim, error) {
	pvcs, err := s.ClientSet.CoreV1().PersistentVolumeClaims(s.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return pvcs.Items, nil
}

func (s *StorageClient) CreateLocalPVC(ctx context.Context, name string, sizeGi int) error {
	storageClass := "local-storage"

	storageQuantity, err := resource.ParseQuantity(fmt.Sprintf("%dGi", sizeGi))
	if err != nil {
		return err
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			StorageClassName: &storageClass,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: storageQuantity,
				},
			},
		},
	}

	_, err = s.ClientSet.CoreV1().PersistentVolumeClaims(s.Namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (s *StorageClient) ExistsPVC(ctx context.Context, name string) (bool, error) {
	_, err := s.ClientSet.CoreV1().PersistentVolumeClaims(s.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *StorageClient) CreateIfNotExistsPVC(ctx context.Context, name string, sizeGi int) error {
	exists, err := s.ExistsPVC(ctx, name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.CreateLocalPVC(ctx, name, sizeGi)
}

func (s *StorageClient) DeletePVC(ctx context.Context, name string) error {
	err := s.ClientSet.CoreV1().PersistentVolumeClaims(s.Namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func (s *StorageClient) GetPVC(ctx context.Context, name string) (*corev1.PersistentVolumeClaim, error) {
	pvc, err := s.ClientSet.CoreV1().PersistentVolumeClaims(s.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, fmt.Errorf("PVC %s not found", name)
		}
		return nil, err
	}
	return pvc, nil
}

func (s *StorageClient) ExecuteWithPVC(ctx context.Context, pvcName string) error {
	config := config.GetConfig()
	name := fmt.Sprintf("executor-%s", pvcName)
	runAsUser := int64(1000)
	runAsGroup := int64(1000)
	fsGroup := int64(1000)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": name,
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser:  &runAsUser,
				RunAsGroup: &runAsGroup,
				FSGroup:    &fsGroup,
			},
			Containers: []corev1.Container{
				{
					Name:            "executor",
					Image:           "jliotorresmoreno/odoo-executor:v1.0.0",
					ImagePullPolicy: corev1.PullAlways,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "data",
							MountPath: "/data",
						},
					},
					Env: []corev1.EnvVar{
						{Name: "ADMIN_URL", Value: config.AdminURL},
						{Name: "ADMIN_PASSWORD", Value: config.AdminPassword},
						{Name: "NAMESPACE", Value: s.Namespace},
					},
					Ports: []corev1.ContainerPort{
						{ContainerPort: 4080},
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "data",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			},
		},
	}

	_, err := s.ClientSet.CoreV1().Pods(s.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creando pod: %w", err)
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": name,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       4080,
					TargetPort: intstr.FromInt(4080),
				},
			},
		},
	}

	// Esperar a que el pod esté listo
	watcher, err := s.ClientSet.CoreV1().Pods(s.Namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", name),
	})
	if err != nil {
		return fmt.Errorf("error creando watcher: %w", err)
	}
	defer watcher.Stop()

	for event := range watcher.ResultChan() {
		p, ok := event.Object.(*corev1.Pod)
		if !ok {
			continue
		}
		switch p.Status.Phase {
		case corev1.PodRunning, corev1.PodSucceeded:
			_, err = s.ClientSet.CoreV1().Services(s.Namespace).Create(ctx, service, metav1.CreateOptions{})
			if err != nil && !k8serrors.IsAlreadyExists(err) {
				return fmt.Errorf("error creando servicio: %w", err)
			}

			return nil
		case corev1.PodFailed:
			return fmt.Errorf("el pod falló: %s", p.Status.Reason)
		}
	}

	return fmt.Errorf("el pod nunca llegó a estado Running o Succeeded")
}
