/*
Copyright ApeCloud Inc.

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

package cluster

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/templates"

	"github.com/apecloud/kubeblocks/internal/dbctl/delete"
	"github.com/apecloud/kubeblocks/internal/dbctl/types"
	"github.com/apecloud/kubeblocks/internal/dbctl/util/builder"
)

var deleteExample = templates.Examples(`
		# delete a cluster named my-cluster
		dbctl cluster delete my-cluster
`)

func NewDeleteCmd(f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	return builder.NewCmdBuilder().
		Use("delete").
		Short("Delete a cluster").
		Example(deleteExample).
		GroupKind(types.ClusterGK()).
		Factory(f).
		IOStreams(streams).
		Build(delete.Build)
}
