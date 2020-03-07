
job "scheduler" {
  datacenters = ["dc1"]

  type = "service"

  update {
    stagger     = "5s"
    auto_revert = true
  }

  group "scheduler" {
    count = 1

    task "scheduler" {
      driver = "docker"

      config {
        image = "crccheck/hello-world"
        port_map {
          http = 8000
        }
      }


      service {
        name = "scheduler"
        port = "http"
      }

      resources {
        cpu    = 500
        memory = 256

        network {
          mbits = 1

          port "http" {}
        }
      }
    }
  }
}
