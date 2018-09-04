package spotguide

import (
	"testing"

	"gopkg.in/yaml.v2"
)

func TestUnmarshalDroneRepoConfig(t *testing.T) {
	configYaml := `
pipeline:
  create_cluster:
    image: banzaicloud/plugin-pipeline-client:0.3.0
    cluster_name: "[[your-cluster-name]]"
    cluster_provider: "amazon"
    amazon_node_mincount: "2"

    secrets: [plugin_endpoint, plugin_token]

  install_monitoring:
    image: banzaicloud/plugin-pipeline-client:0.3.0
    deployment_name: "banzaicloud-stable/pipeline-cluster-monitor"
    deployment_release_name: "monitor"

    secrets: [plugin_endpoint, plugin_token]

  install_spark_resources:
    image: banzaicloud/plugin-pipeline-client:0.3.0

    deployment_name: "banzaicloud-stable/spark"
    deployment_release_name: "release-1"
    deployment_values:
      historyServer:
        enabled: true
      spark-hs:
        app:
          logDirectory: "s3a://[[your-s3-bucket]]/"

    secrets: [plugin_endpoint, plugin_token]

  remote_checkout:
    image: banzaicloud/plugin-k8s-proxy:0.3.0
    original_image: plugins/git

  remote_build:
    image: banzaicloud/plugin-k8s-proxy:0.3.0
    original_image: denvazh/scala:2.11.8
    original_commands:
      - sbt clean package

  run:
    image: banzaicloud/plugin-k8s-proxy:0.3.0
    original_image: banzaicloud/plugin-spark-submit-k8s:0.3.0
    proxy_service_account: spark

    spark_submit_options:
      class: com.banzaicloud.sfdata.SFPDIncidents
      kubernetes-namespace: default
      packages: com.typesafe.scala-logging:scala-logging_2.11:3.1.0,ch.qos.logback:logback-classic:1.1.2
    spark_submit_configs:
      spark.app.name: SFPDIncidents
      spark.local.dir: /tmp/spark-locals
      spark.kubernetes.driver.docker.image: banzaicloud/spark-driver:v2.2.1-k8s-1.0.11
      spark.kubernetes.executor.docker.image: banzaicloud/spark-executor:v2.2.1-k8s-1.0.11
      spark.kubernetes.initcontainer.docker.image: banzaicloud/spark-init:v2.2.1-k8s-1.0.11
      spark.dynamicAllocation.enabled: "true"
      spark.kubernetes.resourceStagingServer.uri: http://spark-rss:10000
      spark.kubernetes.resourceStagingServer.internal.uri: http://spark-rss:10000
      spark.shuffle.service.enabled: "true"
      spark.kubernetes.shuffle.namespace: default
      spark.kubernetes.shuffle.labels: app=spark-shuffle-service,spark-version=2.2.0
      spark.kubernetes.authenticate.driver.serviceAccountName: spark
      spark.metrics.conf: /opt/spark/conf/metrics.properties
      spark.eventLog.enabled: "true"
      spark.eventLog.dir: "s3a://[[your-s3-bucket]]/"
    spark_submit_app_args:
      - target/scala-2.11/sf-police-incidents_2.11-0.1.jar
      - --dataPath s3a://[[your-s3-bucket-with-pdi-data]]/Police_Department_Incidents.csv
`
	conifg := droneRepoConfig{}
	err := yaml.Unmarshal([]byte(configYaml), &conifg)

	if err != nil {
		t.Errorf("Unmarshal expected to succeed but got error: %s", err.Error())
	}

}
