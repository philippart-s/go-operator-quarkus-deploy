# go-operator-quarkus-deploy
Operator written in Go to deploy a Quarkus application

## 🎉 Init project
 - la branche `01-init-project` contient le résultat de cette étape
 - [installer / mettre](https://sdk.operatorframework.io/docs/installation/) à jour la dernière version du [Operator SDK](https://sdk.operatorframework.io/) (v1.27.0 au moment de l'écriture du readme)
 - créer le répertoire `go-operator-quarkus-deploy`
 - dans le répertoire `go-operator-quarkus-deploy`, scaffolding du projet : `operator-sdk init --domain wilda.fr --repo github.com/philippart-s/go-operator-quarkus-deploy`
 - A ce stage une arborescence complète a été générée, notamment la partie configuration dans `config` et un `Makefile` permettant le lancement des différentes commandes de build
 - vérification que cela compile : `go build`
