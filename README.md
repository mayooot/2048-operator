Use K8s Operator to run 2048 game.


Command:

~~~shell
kubebuilder init --domain mayooot.github.io --repo mayooot.github.io/2048-operator

kubebuilder create api --group myapp --version v1 --kind Game
~~~