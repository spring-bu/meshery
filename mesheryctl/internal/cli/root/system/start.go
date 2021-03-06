// Copyright 2019 The Meshery Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package system

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"runtime"

	"github.com/layer5io/meshery/mesheryctl/pkg/utils"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	skipUpdateFlag bool
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start Meshery",
	Long:  `Run 'docker-compose' to start Meshery and each of its service mesh adapters.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		//Check prerequisite
		utils.PreReqCheck()

		if _, err := os.Stat(utils.MesheryFolder); os.IsNotExist(err) {
			if err := os.Mkdir(utils.MesheryFolder, 0777); err != nil {
				log.Fatal(err)
			}
		}

		if _, err := os.Stat(utils.DockerComposeFile); os.IsNotExist(err) {
			if err := utils.DownloadFile(utils.DockerComposeFile, fileURL); err != nil {
				log.Fatal("start cmd: ", err)
			}
		}

		//////// FLAGS
		// Control whether to pull for new Meshery container images
		if skipUpdateFlag {
			log.Info("Skipping Meshery update...")
		} else {
			updateMesheryContainers()
		}

		// Reset Meshery config file to default settings
		if utils.ResetFlag {
			resetMesheryConfig()
		}

		log.Info("Starting Meshery...")
		start := exec.Command("docker-compose", "-f", utils.DockerComposeFile, "up", "-d")
		start.Stdout = os.Stdout
		start.Stderr = os.Stderr

		if err := start.Run(); err != nil {
			log.Fatal("Error starting meshery:", err)
			return
		}
		checkFlag := 0 //flag to check

		//connection to docker-client
		cli, err := client.NewEnvClient()
		if err != nil {
			log.Fatal(err)
		}

		containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
		if err != nil {
			log.Fatal(err)
		}

		//check for container meshery_meshery_1 running status
		for _, container := range containers {
			if container.Names[0] == "/meshery_meshery_1" {
				log.Info("Opening Meshery in your browser. If Meshery does not open, please point your browser to http://localhost:9081 to access Meshery.")

				//check for os of host machine
				if runtime.GOOS == "windows" {
					// Meshery running on Windows host
					err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
					if err != nil {
						log.Fatal(err)
					}
				} else if runtime.GOOS == "linux" {
					// Meshery running on Linux host
					err = exec.Command("xdg-open", url).Start()
					if err != nil {
						log.Fatal(err)
					}
				} else {
					// Assume Meshery running on MacOS host
					err = exec.Command("open", url).Start()
					if err != nil {
						log.Fatal(err)
					}
				}

				//check flag to check successful deployment
				checkFlag = 0
				break
			} else {
				checkFlag = 1
			}
		}

		//if meshery_meshery_1 failed to start showing logs
		//code for logs
		if checkFlag == 1 {
			log.Info("Starting Meshery logging . . .")
			cmdlog := exec.Command("docker-compose", "-f", utils.DockerComposeFile, "logs", "-f")
			cmdReader, err := cmdlog.StdoutPipe()
			if err != nil {
				log.Fatal(err)
			}
			scanner := bufio.NewScanner(cmdReader)
			go func() {
				for scanner.Scan() {
					log.Println(scanner.Text())
				}
			}()
			if err := cmdlog.Start(); err != nil {
				log.Fatal(err)
			}
			if err := cmdlog.Wait(); err != nil {
				log.Fatal(err)
			}
		}

	},
}

func init() {
	startCmd.Flags().BoolVarP(&skipUpdateFlag, "skip-update", "", false, "(optional) skip checking for new Meshery's container images.")
	startCmd.Flags().BoolVarP(&utils.ResetFlag, "reset", "", false, "(optional) reset Meshery's configuration file to default settings.")
}
