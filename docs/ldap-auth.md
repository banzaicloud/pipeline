## LDAP authentication setup

### Developer example

The local developer version of Pipeline is already configured to connect to an example LDAP server. This LDAP server is configured in `docker-compose.override.yml.dist` it is just commented out. If you wish to enable it please un-comment the `ldap` and `ldap-config` services and the `ldap-config` volume and start them up.
