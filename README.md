<p align="center">
<img with="290" height="233" src="https://s3.amazonaws.com/dsc-misc/img/acyl.png" alt="Acyl chloride" />
</p>

# Acyl
[![CircleCI](https://circleci.com/gh/dollarshaveclub/acyl.svg?style=svg&circle-token=e02d60ca20107ad11a982978184233225c909541)](https://circleci.com/gh/dollarshaveclub/acyl)
[![Docker Repository on Quay](https://quay.io/repository/dollarshaveclub/acyl/status?token=7750d7c3-5f1f-4d90-82c4-6b32dbd591a1 "Docker Repository on Quay")](https://quay.io/repository/dollarshaveclub/acyl)

*Testing Environments On Demand*


Acyl is a server and CLI utility for creating dynamic testing environments on demand in Kubernetes, triggered by GitHub Pull Requests.

Environments are defined by `acyl.yml` in your source code repository, and consist of one or more Helm Charts that are installed into a new Kubernetes namespace that is created on demand and torn down when you're finished. Environment lifecycles are tied to Pull Requests: a new environment is created when you open a PR, it is updated as you push additional commits to the PR (including force pushes), and finally it is destroyed when the PR is closed or merged, automatically.

Acyl includes features to make team collaboration and environment configuration easier for multiple teams working on complex application stacks, allowing teams to maintain separate isolated testing environments but share revisions of in-progress code when needed.

Acyl has been used in various forms as part of the core Dollar Shave Club software delivery pipeline since 2016, as described in a recent [blog post](https://engineering.dollarshaveclub.com/qa-environments-on-demand-with-kubernetes-5a571b4e273c).

## Web UI

Acyl includes a full web UI with authentication/authorization based upon [GitHub Apps](https://developer.github.com/apps/).

<p align="center">
  <img src="https://dsc-misc.s3.amazonaws.com/acyl-web-ui-event.png" width="300">
  <img src="https://dsc-misc.s3.amazonaws.com/acyl-web-ui-home.png" width="300">
  <img src="https://dsc-misc.s3.amazonaws.com/acyl-web-ui-env.png" width="300">
</p>

## Environment Configuration

Environments are defined by `acyl.yml`, which describes the required Helm Charts along with their release value configuration and the dependency relationships among them. The config file can be thought of as a "Helm compose", analagous to Docker Compose except using Helm Charts instead of individual containers. Acyl uses [Metahelm](https://github.com/dollarshaveclub/metahelm) to construct a dependency graph of the environment charts and installs them in optimal reverse-dependency order.

An `acyl.yml` in one application repository can reference `acyl.yml` files in other repositories, and those applications (and their dependencies) will be transitively included in the environment. In this way complex application stacks maintained by different teams can share testing environment configuration.

### Examples

- Acyl is self-hosting: we use it to create testing environments for Acyl development itself. See [acyl.yml](https://github.com/dollarshaveclub/acyl/blob/master/acyl.yml) in this repository.
- [Furan](https://github.com/dollarshaveclub/furan) also uses Acyl for [testing environments](https://github.com/dollarshaveclub/furan/blob/master/acyl.yml).

## Local Development

Acyl has CLI tools available to visualize, debug and test environment configurations locally.

`acyl config info` will validate, analyze and show a visualization for the acyl.yml in the current directory (which must be a valid git repository with GitHub remotes).

`acyl config test <create/update/delete>` will simulate PR open/push/close events and create, update or delete environments in a local Kubernetes cluster (Docker For Mac Kubernetes, MicroK8s, etc).

For more details, see [Local Development](https://github.com/dollarshaveclub/acyl/wiki/Local-Development).

## Architecture

![Architecture](doc/acyl_architecture.png?raw=true)

- Acyl: This is the server application which listens for GitHub webhook events and performs operations to create, update or destroy environments within your Kubernetes cluster. The server is intended to run within the same cluster, with ClusterAdmin permissions. The Acyl binary also can be used as a local CLI utility for local environment development and debugging.
- Postgres: This is the primary datastore for Acyl.
- [Furan](https://github.com/dollarshaveclub/furan): This is used to build and push application Docker images on demand.
- (*OPTIONAL*) Notifications can be sent to Slack channels or individual users when environments are created or altered.
- (*OPTIONAL*) [Vault](https://www.vaultproject.io/) can be used for Acyl secrets like GitHub tokens and database credentials.

## Further Reading
- [User Guide](https://github.com/dollarshaveclub/acyl/wiki/User-Guide)
- [acyl.yml v2 Specification](https://github.com/dollarshaveclub/acyl/wiki/Acyl.yml-V2-Specification)


## ADMIN GUIDE

### Create acyl deployment with acyl itself
- `GITHUB_TOKEN=*********** acyl config test create --image-build-mode=none -v --search-paths $HOME/Projects/acyl
- Create github application, setting up oauth and github permissions:
  * Contents: Read-only
  * Metadata: Read-only
  * Pull Requets: Read-only
  * Webhooks: Read-only
  * Commit statuses: Read and write
  * Members: Read-only
  * events: pull request pull request review, pull request review comment
- Create and get secrets from github app. Apply to dummy-acyl-secrets secret.
- Generate github token to access repo and add it to furan dummy secrets
### Run DB migrations

On one terminal forward db port to your computer:
- `kubectl -n=<nm> port-forward postgresql-postgresql-0 5432:5432`
On another, run migrations with acyl cli:
- `acyl pg-migrate --postgres-uri "postgresql://postgres:root@localhost:5432/acyl?sslmode=disable"`

### Add neccessary rbac

Example:
```
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: acyl
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: nitro
  namespace: <nm>
```