helm repo add todolist https://paragor.github.io/todolist
helm repo update
helm install my-todo todolist/todolist --version 0.0.3
