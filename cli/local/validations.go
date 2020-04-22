/*
Copyright 2020 Cortex Labs, Inc.

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

package local

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

type ProjectFiles struct {
	ProjectFiles []string // make sure it is absolute paths
}

func (projectFiles ProjectFiles) GetAllPaths() []string {
	return projectFiles.ProjectFiles
}

func (projectFiles ProjectFiles) GetFile(fileName string) ([]byte, error) {
	absPath, err := files.GetAbsPath(fileName)
	if err != nil {
		return nil, err
	}
	for _, path := range projectFiles.ProjectFiles {
		if path == absPath {
			bytes, err := files.ReadFileBytes(absPath)
			if err != nil {
				return nil, err
			}
			return bytes, nil
		}
	}

	return nil, files.ErrorFileDoesNotExist(fileName)
}

func ValidateLocalAPIs(apis []userconfig.API, projectFiles ProjectFiles, awsClient *aws.Client) error {
	if len(apis) == 0 {
		return spec.ErrorNoAPIs()
	}

	warningFlag := false
	apiPortMap := map[int]string{}
	for i := range apis {
		api := &apis[i]
		if err := spec.ValidateAPI(api, projectFiles, types.LocalProviderType, awsClient); err != nil {
			return err
		}
		if api.LocalPort != nil {
			if collidingAPIName, ok := apiPortMap[*api.LocalPort]; ok {
				return errors.Wrap(ErrorDuplicateLocalPort(collidingAPIName), api.Identify(), userconfig.LocalPortKey, s.Int(*apis[i].LocalPort))
			}
			apiPortMap[*api.LocalPort] = api.Name
		}

		if api.Endpoint != nil || api.Autoscaling != nil || api.Tracker != nil || api.UpdateStrategy != nil {
			if !warningFlag {
				fmt.Println(fmt.Sprintf("warning: %s, %s, %s, and %s keys will be ignored because they are not supported in local environment\n", userconfig.EndpointKey, userconfig.AutoscalingKey, userconfig.TrackerKey, userconfig.UpdateStrategyKey))
			}
			warningFlag = true
		}
	}

	dups := spec.FindDuplicateNames(apis)
	if len(dups) > 0 {
		return spec.ErrorDuplicateName(dups)
	}

	return nil
}
