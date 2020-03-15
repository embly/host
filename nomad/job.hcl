
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
        image = "maxmcd/hello:latest"
        port_map {
          http = 8080
        }
        network_mode = "create_shared"
        network_aliases = [
          "scheduler"
        ]
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
    task "scheduler2" {
      driver = "docker"

      config {
        image = "maxmcd/hello:latest"
        port_map {
          http = 8080
        }
        network_mode = "pricing_db_app_default"
        network_aliases = [
          "scheduler2"
        ]
      }


      service {
        name = "scheduler2"
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
