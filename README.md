# Deedeeploy

Deedeeploy aims to make a developers life a little easier by providing a simple
command-line script to deploy code to a host.

## Access to target host(s)

Before deploying, the host should be configured to allow automated deploys. The
recommended way for deedeeploy to function optimal is as follows.

* Create a *deploy* user on the target host
* Add your public SSH key to the authorized_hosts file of the *deploy* user on
 the target host.
* Make sure the deploy user has read-write access to the target directory
* Check if git or subversion is installed on the target host
