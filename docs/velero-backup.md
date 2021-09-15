## How to test basic backup functionality

In case you have to update Velero helm chart, these are the steps to test basic backup functionality:

>Prequisite: Download the [Velero CLI](https://github.com/vmware-tanzu/velero/releases/) same version as [Velero server image](https://github.com/banzaicloud/pipeline/blob/c4d426c4597770a799faf9cf2e59ebd09f3f2ac0/internal/cmd/config.go#L810).

1. Start a command shell with configured kubeconfig for your cluster:

    ```bash
    banzai cluster --cluster-name YOUR_CLUSTER_NAME shell

    INFO[0001] Running /bin/zsh
    ```

2. Install Wordpress or some example app which is creates persistent volumes:

    ```bash
    helm repo add bitnami https://charts.bitnami.com/bitnami
    helm install wordpress bitnami/wordpress
    ```

3. Make sure persistent volumes were created:

    ```bash
    kubectl get pv

    NAME                                       CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM                              STORAGECLASS   REASON   AGE
    pvc-44d380a1-556a-4834-a39d-f4f325b5f894   8Gi        RWO            Delete           Bound    default/data-wordpress-mariadb-0   gp2                     2s
    pvc-be31fdfd-ec26-4744-8af5-2378c2dce476   10Gi       RWO            Delete           Bound    default/wordpress                  gp2                     2s
    ```

4. Enable Backup service for cluster:

    ```bash
    banzai cluster --cluster-name YOUR_CLUSTER_NAME service backup enable

    ? Schedule backups for every daily
    ? Keep backups for 1 day
    ? Select storage provider: Amazon S3
    ? Provider secret: aws-scratch-s3
    ? Bucket name: {YOUR_S3_BUCKET_NAME}
    ? Service Account Role ARN to use for Velero
    ? Use provider secret to give access for Velero to make volume snapshots Yes
    INFO[0019] Enabling backup service for [100] cluster
    INFO[0070] Backup service is enabled for [100] cluster
    ```

5. Check Velero deployment is running and backup resource has been created:

    ```bash
    kubectl get deployment -n pipeline-system

    NAME                                READY   UP-TO-DATE   AVAILABLE   AGE
    ...
    velero                              1/1     1            1           6m15s

    kubectl get backup -n pipeline-system

    NAME                           AGE
    h1vmlol1awy8a-20210910122426   158m
    ```

6. List backups with Velero CLI:

    ```bash
    velero backup get -n pipeline-system

    NAME                           STATUS      ERRORS   WARNINGS   CREATED                          EXPIRES   STORAGE LOCATION   SELECTOR
    YOUR_CLUSTER_NAME-20210910122426   Completed   0        0          2021-09-10 14:24:26 +0200 CEST   21h       default            <none>
    ```

7. Check that logs are available:

    ```bash
    velero backup logs YOUR_CLUSTER_NAME-20210910122426  -n pipeline-syste
    ```

You may also check that backup log and content files are available on S3 bucket you have specified at step 4.
