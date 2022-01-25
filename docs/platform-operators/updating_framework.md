# Updating a8s Data Services on Kubernetes

Currently we don’t formally version the a8s Data Services on Kubernetes framework. New versions of 
a8s Data Services on Kubernetes are simply represented by new commits to the main branch of this
repository. We plan to switch to a more formal versioning and releasing strategy in the future
(e.g. periodic releases with a semver 2 version).

So if you always want to have the most recent version of a8s Data Services on Kubernetes running in
your cluster, you can just monitor the main branch of this repository and re-execute the manual
installation steps described there whenever there’s a new commit. The installation instructions are
idempotent so if you re-execute them whatever can be updated will be updated but things that haven’t
changed will be left running without disruption. This also means you don’t have to uninstall an old
version of a8s Data Services on Kubernetes before updating to a new version. 

If you want, you can also set up a GitOps solution (such as ArgoCD) to monitor the main branch of
this repository and update your installation of a8s Data Services on Kubernetes automatically,
without manual intervention. However, this can be done only for automatically updating the a9s
Data Services on Kubernetes framework, not for initial installation. That is, there’s an
assumption that the framework has been already installed manually in the cluster (there are
some installation steps that involve credentials that right now we can’t fully automate via GitOps).

We will try not to cause disruption to already running Data Service Instances (DSIs) when upgrading
the a8s control plane, but can make no hard promise at this point in time. If an upgrade has
disruptive consequences for already running DSIs, we'll provide guidance on how to handle the
upgrade when releasing it.
