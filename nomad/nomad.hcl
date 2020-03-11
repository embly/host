datacenter = "dc1"

region = "dc1-1"

data_dir = "/nomad/data/"

bind_addr = "{{ GetPrivateIP }}"

advertise {
  http = "{{ GetPrivateIP }}:4646"
  rpc  = "{{ GetPrivateIP }}:4647"
  serf = "{{ GetPrivateIP }}:4648"
}

consul {
  address = "consul:8500"
}
