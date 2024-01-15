/*
Copyright (C) 2022-2023 ApeCloud Co., Ltd

This file is part of KubeBlocks project

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package cluster

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1alpha1 "github.com/apecloud/kubeblocks/apis/apps/v1alpha1"
	"github.com/apecloud/kubeblocks/pkg/configuration/core"

	"github.com/apecloud/kbcli/pkg/types"
	"github.com/apecloud/kbcli/pkg/util"
)

type configSpecsType []*configSpecMeta

type configSpecMeta struct {
	Spec      appsv1alpha1.ComponentTemplateSpec
	ConfigMap *corev1.ConfigMap

	ConfigSpec       *appsv1alpha1.ComponentConfigSpec
	ConfigConstraint *appsv1alpha1.ConfigConstraint
}

type ConfigRelatedObjects struct {
	Cluster        *appsv1alpha1.Cluster
	ClusterDef     *appsv1alpha1.ClusterDefinition
	ClusterVersion *appsv1alpha1.ClusterVersion
	Comps          []*appsv1alpha1.Component
	CompDefs       []*appsv1alpha1.ComponentDefinition
	ConfigSpecs    map[string]configSpecsType
}

type configObjectsWrapper struct {
	namespace   string
	clusterName string
	components  []string

	err error
	cli dynamic.Interface
}

func (c configSpecsType) findByName(name string) *configSpecMeta {
	for _, spec := range c {
		if spec.Spec.Name == name {
			return spec
		}
	}
	return nil
}

func (c configSpecsType) listConfigSpecs(ccFilter bool) []string {
	var names []string
	for _, spec := range c {
		if spec.ConfigSpec != nil && (!ccFilter || spec.ConfigConstraint != nil) {
			names = append(names, spec.Spec.Name)
		}
	}
	return names
}

func New(clusterName string, namespace string, cli dynamic.Interface, component ...string) *configObjectsWrapper {
	return &configObjectsWrapper{namespace, clusterName, component, nil, cli}
}

func (w *configObjectsWrapper) GetObjects() (*ConfigRelatedObjects, error) {
	objects := &ConfigRelatedObjects{}
	err := w.cluster(objects).
		clusterDefinition(objects).
		clusterVersion(objects).
		compDefs(objects).
		comps(objects).
		configSpecsObjects(objects).
		finish()
	if err != nil {
		return nil, err
	}
	return objects, nil
}

func (w *configObjectsWrapper) configMap(specName string, component string, out *configSpecMeta) *configObjectsWrapper {
	fn := func() error {
		key := client.ObjectKey{
			Namespace: w.namespace,
			Name:      core.GetComponentCfgName(w.clusterName, component, specName),
		}
		out.ConfigMap = &corev1.ConfigMap{}
		return util.GetResourceObjectFromGVR(types.ConfigmapGVR(), key, w.cli, out.ConfigMap)
	}
	return w.objectWrapper(fn)
}

func (w *configObjectsWrapper) configConstraint(specName string, out *configSpecMeta) *configObjectsWrapper {
	fn := func() error {
		if specName == "" {
			return nil
		}
		key := client.ObjectKey{
			Namespace: "",
			Name:      specName,
		}
		out.ConfigConstraint = &appsv1alpha1.ConfigConstraint{}
		return util.GetResourceObjectFromGVR(types.ConfigConstraintGVR(), key, w.cli, out.ConfigConstraint)
	}
	return w.objectWrapper(fn)
}

func (w *configObjectsWrapper) cluster(objects *ConfigRelatedObjects) *configObjectsWrapper {
	fn := func() error {
		clusterKey := client.ObjectKey{
			Namespace: w.namespace,
			Name:      w.clusterName,
		}
		objects.Cluster = &appsv1alpha1.Cluster{}
		if err := util.GetResourceObjectFromGVR(types.ClusterGVR(), clusterKey, w.cli, objects.Cluster); err != nil {
			return makeClusterNotExistErr(w.clusterName)
		}
		return nil
	}
	return w.objectWrapper(fn)
}

func (w *configObjectsWrapper) clusterVersion(objects *ConfigRelatedObjects) *configObjectsWrapper {
	fn := func() error {
		clusterVerName := objects.Cluster.Spec.ClusterVersionRef
		if clusterVerName == "" {
			return nil
		}
		clusterVerKey := client.ObjectKey{
			Namespace: "",
			Name:      clusterVerName,
		}
		objects.ClusterVersion = &appsv1alpha1.ClusterVersion{}
		return util.GetResourceObjectFromGVR(types.ClusterVersionGVR(), clusterVerKey, w.cli, objects.ClusterVersion)
	}
	return w.objectWrapper(fn)
}

func (w *configObjectsWrapper) comps(object *ConfigRelatedObjects) *configObjectsWrapper {
	fn := func() error {
		if len(object.CompDefs) == 0 {
			return nil
		}
		object.Comps = make([]*appsv1alpha1.Component, len(object.Cluster.Spec.ComponentSpecs))
		for i, comp := range object.Cluster.Spec.ComponentSpecs {
			if comp.ComponentDef == "" {
				continue
			}
			compKey := client.ObjectKey{
				Namespace: w.namespace,
				Name:      fmt.Sprintf("%s-%s", w.clusterName, comp.Name),
			}
			temp := appsv1alpha1.Component{}
			err := util.GetResourceObjectFromGVR(types.ComponentGVR(), compKey, w.cli, &temp)
			if err != nil {
				return err
			}
			object.Comps[i] = &temp
		}
		return nil
	}
	return w.objectWrapper(fn)
}

func (w *configObjectsWrapper) compDefs(object *ConfigRelatedObjects) *configObjectsWrapper {
	fn := func() error {
		object.CompDefs = make([]*appsv1alpha1.ComponentDefinition, 0)
		for _, comp := range object.Cluster.Spec.ComponentSpecs {
			if comp.ComponentDef == "" {
				continue
			}
			compDefKey := client.ObjectKey{
				Namespace: "",
				Name:      comp.ComponentDef,
			}
			temp := appsv1alpha1.ComponentDefinition{}
			err := util.GetResourceObjectFromGVR(types.CompDefGVR(), compDefKey, w.cli, &temp)
			if err != nil {
				return err
			}
			object.CompDefs = append(object.CompDefs, &temp)
		}
		return nil
	}
	return w.objectWrapper(fn)
}

func (w *configObjectsWrapper) clusterDefinition(objects *ConfigRelatedObjects) *configObjectsWrapper {
	fn := func() error {
		clusterVerKey := client.ObjectKey{
			Namespace: "",
			Name:      objects.Cluster.Spec.ClusterDefRef,
		}
		objects.ClusterDef = &appsv1alpha1.ClusterDefinition{}
		return util.GetResourceObjectFromGVR(types.ClusterDefGVR(), clusterVerKey, w.cli, objects.ClusterDef)
	}
	return w.objectWrapper(fn)
}

func (w *configObjectsWrapper) configSpecsObjects(objects *ConfigRelatedObjects) *configObjectsWrapper {
	fn := func() error {
		components := w.components
		if len(components) == 0 {
			components = getComponentNames(objects.Cluster)
		}
		configSpecs := make(map[string]configSpecsType, len(components))
		for i, component := range components {
			if configSpecs[component] == nil {
				configSpecs[component] = make(configSpecsType, 0)
			}
			// if the object have the new API
			if len(objects.CompDefs) != 0 {
				componentConfigSpecs, err := w.genConfigSpecsByCompDef(objects.Comps[i], objects.CompDefs)
				if err != nil {
					return err
				}
				configSpecs[component] = append(configSpecs[component], componentConfigSpecs...)
				componentScriptsSpecs, err := w.genScriptsSpecsByCompDef(objects.Comps[i], objects.CompDefs)
				if err != nil {
					return err
				}
				configSpecs[component] = append(configSpecs[component], componentScriptsSpecs...)
			} else {
				componentConfigSpecs, err := w.genConfigSpecs(objects, component)
				if err != nil {
					return err
				}
				configSpecs[component] = append(configSpecs[component], componentConfigSpecs...)
				componentScriptsSpecs, err := w.genScriptsSpecs(objects, component)
				if err != nil {
					return err
				}
				configSpecs[component] = append(configSpecs[component], componentScriptsSpecs...)
			}

		}
		objects.ConfigSpecs = configSpecs
		return nil
	}
	return w.objectWrapper(fn)
}

func (w *configObjectsWrapper) finish() error {
	return w.err
}

func (w *configObjectsWrapper) genScriptsSpecs(objects *ConfigRelatedObjects, component string) ([]*configSpecMeta, error) {
	cComponent := objects.Cluster.Spec.GetComponentByName(component)
	if cComponent == nil {
		return nil, core.MakeError("not found component %s in cluster %s", component, objects.Cluster.Name)
	}
	dComponent := objects.ClusterDef.GetComponentDefByName(cComponent.ComponentDefRef)
	if dComponent == nil {
		return nil, core.MakeError("not found component %s in cluster definition %s", component, objects.ClusterDef.Name)
	}
	configSpecMetas := make([]*configSpecMeta, 0)
	for _, spec := range dComponent.ScriptSpecs {
		meta, err := w.transformScriptsSpecMeta(spec, component)
		if err != nil {
			return nil, err
		}
		configSpecMetas = append(configSpecMetas, meta)
	}
	return configSpecMetas, nil
}

func (w *configObjectsWrapper) transformConfigSpecMeta(spec appsv1alpha1.ComponentConfigSpec, component string) (*configSpecMeta, error) {
	specMeta := &configSpecMeta{
		Spec:       spec.ComponentTemplateSpec,
		ConfigSpec: spec.DeepCopy(),
	}
	err := w.configMap(spec.Name, component, specMeta).
		configConstraint(spec.ConfigConstraintRef, specMeta).
		finish()
	if err != nil {
		return nil, err
	}
	return specMeta, nil
}

func (w *configObjectsWrapper) transformScriptsSpecMeta(spec appsv1alpha1.ComponentTemplateSpec, component string) (*configSpecMeta, error) {
	specMeta := &configSpecMeta{
		Spec: spec,
	}
	err := w.configMap(spec.Name, component, specMeta).
		finish()
	if err != nil {
		return nil, err
	}
	return specMeta, nil
}

func (w *configObjectsWrapper) objectWrapper(fn func() error) *configObjectsWrapper {
	if w.err != nil {
		return w
	}
	w.err = fn()
	return w
}

func (w *configObjectsWrapper) genConfigSpecs(objects *ConfigRelatedObjects, component string) ([]*configSpecMeta, error) {
	var (
		ret []*configSpecMeta

		cComponents = objects.Cluster.Spec.ComponentSpecs
		dComponents = objects.ClusterDef.Spec.ComponentDefs
		vComponents []appsv1alpha1.ClusterComponentVersion
	)

	if objects.ClusterVersion != nil {
		vComponents = objects.ClusterVersion.Spec.ComponentVersions
	}
	configSpecs, err := core.GetConfigTemplatesFromComponent(cComponents, dComponents, vComponents, component)
	if err != nil {
		return nil, err
	}
	for _, spec := range configSpecs {
		meta, err := w.transformConfigSpecMeta(spec, component)
		if err != nil {
			return nil, err
		}
		ret = append(ret, meta)
	}
	return ret, nil
}

func (w *configObjectsWrapper) genConfigSpecsByCompDef(comp *appsv1alpha1.Component, compDefs []*appsv1alpha1.ComponentDefinition) ([]*configSpecMeta, error) {
	var (
		ret []*configSpecMeta
	)
	if comp == nil {
		return nil, fmt.Errorf("the component is nil and fail to get the corresponding configs")
	}
	for _, compDef := range compDefs {
		if compDef.Name != comp.Spec.CompDef {
			continue
		}
		for _, spec := range compDef.Spec.Configs {
			specMeta := &configSpecMeta{
				Spec:       spec.ComponentTemplateSpec,
				ConfigSpec: spec.DeepCopy(),
				ConfigMap:  &corev1.ConfigMap{},
			}
			err := util.GetResourceObjectFromGVR(types.ConfigmapGVR(), client.ObjectKey{
				Namespace: w.namespace,
				Name:      fmt.Sprintf("%s-%s", comp.Name, spec.Name),
			}, w.cli, specMeta.ConfigMap)
			if err != nil {
				return nil, err
			}
			if spec.ConfigConstraintRef != "" {
				cc := &appsv1alpha1.ConfigConstraint{}
				err = util.GetResourceObjectFromGVR(types.ConfigConstraintGVR(), client.ObjectKey{
					Namespace: "",
					Name:      spec.ConfigConstraintRef,
				}, w.cli, cc)
				if err != nil {
					return nil, err
				}
				specMeta.ConfigConstraint = cc
			}

			ret = append(ret, specMeta)
		}
		break
	}
	return ret, nil
}

func (w *configObjectsWrapper) genScriptsSpecsByCompDef(comp *appsv1alpha1.Component, compDefs []*appsv1alpha1.ComponentDefinition) ([]*configSpecMeta, error) {
	var (
		ret []*configSpecMeta
	)
	if comp == nil {
		return nil, fmt.Errorf("the component is nil and fail to get the corresponding scripts")
	}
	for _, compDef := range compDefs {
		if compDef.Name != comp.Spec.CompDef {
			continue
		}
		for _, spec := range compDef.Spec.Scripts {
			specMeta := &configSpecMeta{
				Spec:      spec,
				ConfigMap: &corev1.ConfigMap{},
			}
			err := util.GetResourceObjectFromGVR(types.ConfigmapGVR(), client.ObjectKey{
				Namespace: w.namespace,
				Name:      fmt.Sprintf("%s-%s", comp.Name, spec.Name),
			}, w.cli, specMeta.ConfigMap)
			if err != nil {
				return nil, err
			}
			ret = append(ret, specMeta)
		}
		break
	}
	return ret, nil
}
