# init
```
kubebuilder init --domain github.com --repo github.com/tonyshanc/sample-operator-v2
kubebuilder create api --group sample --version v1 --kind At
```
# change default namespace
```
kubectl create ns sample
kubectl config set-context $(kubectl config current-context) --namespace=sample
```

# regist CRD
```
make install
```
# run controller
```
make run
```
if you edit API definitions, generate manifests as CRs or CRDs using
```
make manifests
```

# create CR
```
k apply -f config/samples/sample_v1_car.yaml -n sample
```