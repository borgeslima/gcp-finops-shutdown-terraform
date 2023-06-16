package shutdown

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudevents/sdk-go/v2/event"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/option"

	"google.golang.org/api/container/v1"
)

type PubSubMessage struct {
	Data []byte `json:"data"`
}

func ProcessPubSub(ctx context.Context, eve event.Event) error {

	var msg PubSubMessage

	if err := eve.DataAs(&msg); err != nil {
		return fmt.Errorf("event.DataAs: %w", err)
	}

	name := string(msg.Data)

	fmt.Printf("Iniciando função: %s", name)

	action := string(msg.Data)

	nodeCount := int64(0)

	if action == "reduce" {
		nodeCount = int64(0)
	} else {
		nodeCount = int64(3)
	}

	client, err := cloudresourcemanager.NewService(ctx)

	if err != nil {
		log.Fatalf("Falha ao criar o cliente: %v", err)
	}

	// Chamada para listar projetos na organização principal

	projects, err := client.Projects.List().Do()

	if err != nil {
		log.Fatalf("Falha ao listar projetos: %v", err)
	}

	// Autenticação usando credenciais padrão do ambiente

	clientCluster, err := container.NewService(ctx, option.WithScopes(container.CloudPlatformScope))

	if err != nil {
		log.Fatalf("Falha ao criar o cliente: %v", err)
	}

	// Autenticação usando credenciais padrão do ambiente

	cloudPlatformScope, err := container.NewService(ctx, option.WithScopes(container.CloudPlatformScope))

	if err != nil {
		log.Fatalf("Falha ao criar o cliente: %v", err)
	}

	for _, project := range projects.Projects {

		clusters, err := clientCluster.Projects.Locations.Clusters.List("projects/" + project.ProjectId + "/locations/-").Do()

		if err != nil {
			log.Fatalf("Falha ao listar clusters: %v", err)
		}

		for key, value := range project.Labels {

			if key == "za_environment" && (value == "dev" || value == "hml") {

				for _, cluster := range clusters.Clusters {

					// Verificar se o cluster é Autopilot

					if cluster.Autopilot == nil {

						clusterPath := ("projects/" + project.ProjectId + "/locations/" + cluster.Zone + "/clusters/" + cluster.Name)

						// fmt.Printf("- Ipv4Cidr: (%s) Project: (%s) Cluster:(%s)\n", cluster.ClusterIpv4Cidr, project.Name, cluster.Name)

						nodePools, err := cloudPlatformScope.Projects.Locations.Clusters.NodePools.List(clusterPath).Do()

						if err != nil {
							log.Fatalf("Falha ao obter os node pools do cluster: %v", err)
						}

						fmt.Printf("-------- Atualizando cluster (%s)--------- \n", cluster.Name)

						for _, nodePool := range nodePools.NodePools {

							fmt.Printf("- Nome do nó: %s\n", nodePool.Name)
							fmt.Printf("- Versão do nó : %s\n", nodePool.Version)
							fmt.Printf("- DiskType: %s\n", nodePool.Config.DiskType)
							fmt.Printf("- MachineType: %s\n", nodePool.Config.MachineType)
							fmt.Printf("- Quantidade de nós(%s)\n", nodeCount)

							// Construir a solicitação de atualização do node pool
							request := &container.UpdateNodePoolRequest{
								NodeVersion: nodePool.Version,
							}

							// Realizar a atualização do node pool
							_, err := cloudPlatformScope.Projects.Locations.Clusters.NodePools.Update("projects/"+project.ProjectId+"/locations/"+cluster.Location+"/clusters/"+cluster.Name+"/nodePools/"+nodePool.Name, request).Context(ctx).Do()
							if err != nil {
								continue
							}

						}
					}

				}
			}
		}

	}
	return nil
}
