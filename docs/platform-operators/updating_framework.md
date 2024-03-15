# Updating a8s Data Services on Kubernetes

The installation instructions are idempotent so if you re-execute them whatever can be updated
will be updated but things that haven’t changed will be left running without disruption. This
also means you don’t have to uninstall an old version of a8s Data Services on Kubernetes before
updating to a new version.

We aim to minimize disruption to currently running Data Service Instances (DSIs) during the a8s
control plane upgrade. However, we cannot make a firm commitment at this stage. If the upgrade
poses any disruptive consequences for existing DSIs, we will offer guidance on how to manage the
upgrade when it is released.
