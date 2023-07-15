# visitors-operator
Operator SDK Example from Kubernetes Operators Book from Jason Dobies & Joshua Wood. The xample was updated to reflect latest changes on framework.

# Steps:

1. Create the repository
2. Init project:

```
OPERATOR_NAME=visitors-operator
PROJECT_NAME=visitors-project
export GO111MODULE=on
operator-sdk init --project-name $PROJECT_NAME --domain example.com --repo github.com/Crisarias/visitors-operator
operator-sdk create api --group default --version v1alpha1 --kind VisitorApp --resource --controller
Modify visitorapp_types.go
make
make manifests
```

3. Construct app logic and reconcile method
4. Deploy crd

```
kubectl apply -f ./config/crd/bases/default.example.com_visitorapps
```

5. Deploy or Run operator

```
make run install
```

6. Update sample file and deploy cr

```
kubectl apply -f ./config/samples/default_v1alpha1_visitorapp
```

