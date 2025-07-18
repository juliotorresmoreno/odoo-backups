package storage

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
)

type StorageClient struct {
	ClientSet *k8sclient.Clientset
	Namespace string
}

func NewStorageClient(clientSet *k8sclient.Clientset, namespace string) *StorageClient {
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

func (s *StorageClient) CreatePVC(ctx context.Context, name string, storageClass string, sizeGi int) error {
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
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}

	return nil
}

func (s *StorageClient) CreateIfNotExistsPVC(ctx context.Context, name string, storageClass string, sizeGi int) error {
	_, err := s.ClientSet.CoreV1().PersistentVolumeClaims(s.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !k8serrors.IsNotFound(err) {
		return err
	}
	return s.CreatePVC(ctx, name, storageClass, sizeGi)
}
