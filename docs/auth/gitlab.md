## GitLab OAuth App setup

### Create a personal access token on GitLab

Create a [personal access token](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html) on GitLab.

Take note of the generated GitLab access token as it will be needed.

### Register the OAuth application on GitLab

Register an [OAuth](https://docs.gitlab.com/ee/integration/oauth_provider.html) application on GitLab for the Pipeline API and CI/CD workflow.

![gitlab oauth app reg](images/GitLabOAuthAppReg.png)

Fill in `Authorization callback URL`. This field has to be updated once the Control Plane is up and running using the IP address or the DNS name:

- For local usage:
    ```bash
    http://127.0.0.1:5556/dex/callback
    ```

- For on-cloud usage:
    ```bash
    http://{control_plane_public_address}/dex/callback
    ```

Take note of the `Client ID` and `Client Secret` as these will be required for launching the Pipeline Control Plane and fill them into the `config/dex.yml` file (or into environment variables, see that file for details).

![gitlab oauth app id](images/GitLabOAuthAppID.png)
