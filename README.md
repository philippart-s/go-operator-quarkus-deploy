# go-operator-quarkus-deploy
Operator written in Go to deploy a Quarkus application

## 🎉 Init project
 - la branche `01-init-project` contient le résultat de cette étape
 - [installer / mettre](https://sdk.operatorframework.io/docs/installation/) à jour la dernière version du [Operator SDK](https://sdk.operatorframework.io/) (v1.29.0 au moment de l'écriture du readme)
 - créer le répertoire `go-operator-quarkus-deploy`
 - dans le répertoire `go-operator-quarkus-deploy`, scaffolding du projet : `operator-sdk init --domain wilda.fr --repo github.com/philippart-s/go-operator-quarkus-deploy`
 - A ce stage une arborescence complète a été générée, notamment la partie configuration dans `config` et un `Makefile` permettant le lancement des différentes commandes de build
 - vérification que cela compile : `go build`

 ## 📄 CRD generation
 - la branche `02-crd-generation` contient le résultat de cette étape
 - création de l'API : `operator-sdk create api --version v1 --kind QuarkusOperator --resource --controller`
 - de nouveau, de nombreux fichiers de générés, notamment le controller `./controllers/quarkusoperator_controller.go`
 - ensuite on génère la CRD `./config/crd/bases/wilda.fr_quarkusoperators.yaml` avec la commande `make manifests`
  - puis on peut l'appliquer avec la commande `make install`
  - et vérfier qu'elle a été créée : `kubectl get crd quarkusoperators.wilda.fr`
```bash
$ kubectl get crd quarkusoperators.wilda.fr

NAME                         CREATED AT
quarkusoperators.wilda.fr   2022-09-01T06:43:36Z
```
