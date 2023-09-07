/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	apiv1alpha1 "github.com/nxtcoder19/mongo-operator/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// MongoReconciler reconciles a Mongo object
type MongoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=api.my.domain,resources=mongoes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=api.my.domain,resources=mongoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=api.my.domain,resources=mongoes/finalizers,verbs=update

func (r *MongoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//_ = log.FromContext(ctx)

	// TODO(user): your logic here
	fmt.Printf("hello world from %v\n", req.NamespacedName.String())

	// fetching the instance
	app := &apiv1alpha1.Mongo{}
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// checking StatefulSet exist or we have to create
	foundThis := &v1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, foundThis)
	if err != nil {
		if errors.IsNotFound(err) {
			//define or create a new StatefulSet
			// dep := r(app)
			dep := r.StatefulSetForMongo(app)
			if err = r.Create(ctx, dep); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		} else {
			return ctrl.Result{}, err
		}
	}
	replicas := app.Spec.Size
	if foundThis.Spec.Replicas != &replicas {
		foundThis.Spec.Replicas = &replicas
		err := r.Update(ctx, foundThis)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	//Check if the existing StatefulSet needs an update
	//if r.StatefulSetNeedsUpdate(foundThis, app) {
	//	// Update the existing StatefulSet
	//	updated := r.UpdateStatefulSet(foundThis, app)
	//	if err = r.Update(ctx, updated); err != nil {
	//		return ctrl.Result{}, err
	//	}
	//}

	return ctrl.Result{}, nil
}

// StatefulSetForApp return a StatefulSet object
func (r *MongoReconciler) StatefulSetForMongo(m *apiv1alpha1.Mongo) *v1.StatefulSet {
	lbls := labelsForApp(m.Name)
	replicas := m.Spec.Size
	//cpu := m.Spec.Cpu

	dep := &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: v1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: lbls,
			},
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: lbls,
				},
				Spec: coreV1.PodSpec{
					Containers: []coreV1.Container{{
						Image: "mongo:latest", // MongoDB image
						Name:  "mongo",        // Container name
						Ports: []coreV1.ContainerPort{{
							ContainerPort: 27017, // MongoDB port
							Name:          "mongo",
						}},
						Env: []coreV1.EnvVar{{
							Name: "MONGO_INITDB_ROOT_USERNAME",
							ValueFrom: &coreV1.EnvVarSource{
								SecretKeyRef: &coreV1.SecretKeySelector{
									LocalObjectReference: coreV1.LocalObjectReference{
										Name: "mongo-credentials",
									},
									Key: "username",
								},
							},
						}, {
							Name: "MONGO_INITDB_ROOT_PASSWORD",
							ValueFrom: &coreV1.EnvVarSource{
								SecretKeyRef: &coreV1.SecretKeySelector{
									LocalObjectReference: coreV1.LocalObjectReference{
										Name: "mongo-credentials",
									},
									Key: "password",
								},
							},
						}},
						Resources: coreV1.ResourceRequirements{
							Limits: coreV1.ResourceList{
								coreV1.ResourceCPU: resource.MustParse("20"),
							},
							Requests: coreV1.ResourceList{
								coreV1.ResourceCPU: resource.MustParse("4"),
							},
						},
					}},
				},
			},
		},
	}

	controllerutil.SetControllerReference(m, dep, r.Scheme)
	return dep
}

//func (r *MongoReconciler) UpdateStatefulSet(existing *v1.StatefulSet, desired *apiv1alpha1.Mongo) *v1.StatefulSet {
//	// Make a copy of the existing StatefulSet to update
//	updated := existing.DeepCopy()
//
//	// Update the relevant fields in the StatefulSet based on changes in the Mongo CR
//	updated.Spec.Template.Spec.Containers[0].Resources.Requests[coreV1.ResourceCPU] = resource.MustParse("7")
//
//	return updated
//}

//func (r *MongoReconciler) StatefulSetNeedsUpdate(existing *v1.StatefulSet, desired *apiv1alpha1.Mongo) bool {
//	// Compare the CPU requests in the existing StatefulSet and the desired Mongo CR
//	existingCPURequest := existing.Spec.Template.Spec.Containers[0].Resources.Requests[coreV1.ResourceCPU]
//	desiredCPURequest := resource.MustParse("7")
//	//fmt.Println(fmt.Sprintf("desired cpu request is %v", desiredCPURequest))
//
//	// Check if the CPU requests are different
//	return !existingCPURequest.Equal(desiredCPURequest)
//}

// labelsForApp create a simple sets of labels for App
func labelsForApp(name string) map[string]string {
	return map[string]string{"cr_name": name}
}

// SetupWithManager sets up the controller with the Manager.
func (r *MongoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr)
	builder.For(&apiv1alpha1.Mongo{})
	builder.Owns(&v1.StatefulSet{})
	return builder.Complete(r)
	//return ctrl.NewControllerManagedBy(mgr).
	//	For(&apiv1alpha1.Mongo{}).
	//	Complete(r)
}
