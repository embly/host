


```bash
embly deploy ../pkg/scheduler
embly deploy ../pkg/scheduler:backend
```

```python
load("../pkg/scheduler", "scheduler")
load("../pkg/scheduler:backend", "backend")


load("github.com/maxmcd/examples/flask/scheduler:backend", "backend")
```

notes:
 - think about go.mod/sum or equivalent, with hashes and to determine the project root
 - build proxy, builds projects for pulling, just pull the docker image, could just pull the repo and build locally to begin with
