datacenter = "dc1"

region = "dc1-1"

data_dir = "/nomad/data/"

# bind_addr = "{{ GetPrivateIP }}"
bind_addr = "127.0.0.1"

advertise {
  # http = "{{ GetPrivateIP }}:4646"
  # rpc  = "{{ GetPrivateIP }}:4647"
  # serf = "{{ GetPrivateIP }}:4648"
  http = "127.0.0.1:4646"
  rpc  = "127.0.0.1:4647"
  serf = "127.0.0.1:4648"
}

consul {
  # address = "consul:8500"
  address = "127.0.0.1:8500"
}

client {
  network_interface = "docker0"
}
