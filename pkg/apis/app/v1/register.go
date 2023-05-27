/*
Copyright 2019 Hypo.

Licensed under the GNU General Public License, Version 3 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/Coderhypo/canary-crd/blob/master/LICENSE

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// NOTE: Boilerplate only.  Ignore this file.

// Package v1 contains API Schema definitions for the app v1 API group
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=package,register
// +k8s:conversion-gen=canary-crd/pkg/apis/app
// +k8s:defaulter-gen=TypeMeta
// +groupName=app.o0w0o.cn
package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/runtime/scheme"
)

//这个文件用于注册上述定义的数据结构到 Kubernetes 的 schema 中，使得 Kubernetes 能够识别和处理这些自定义资源。

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: "app.o0w0o.cn", Version: "v1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}

	// AddToScheme is required by pkg/client/...
	AddToScheme = SchemeBuilder.AddToScheme
)

// Resource is required by pkg/client/listers/...
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}
