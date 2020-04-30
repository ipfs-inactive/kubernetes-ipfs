variable ssh_authorized_keys_github {
  type = "list"
}

resource "null_resource" "authorized_keys" {
  count      = "${var.count}"

  depends_on = [
    "null_resource.kubernetes"
  ]

  triggers {
    ip    = "${element(var.vpn_ips, 0)}"
    users = "${join(" ", var.ssh_authorized_keys_github)}"
  }

  connection {
    host  = "${element(var.connections, count.index)}"
    user  = "root"
    agent = true
  }

  provisioner "remote-exec" {
    inline = <<EOT
  (
  set -x;
  for USER in ${join(" ", var.ssh_authorized_keys_github)}; do
    curl --fail --max-time 10 "https://github.com/$${USER}.keys" || {
      sleep 2;
      curl --fail --max-time 10 "https://github.com/$${USER}.keys";
    } || exit 1;
  done
  ) >> ~/.ssh/authorized_keys

  STATUS=$$?

  # dedupe
  awk '!x[$0]++' ~/.ssh/authorized_keys \
    > ~/.ssh/authorized_keys.new \
    && mv ~/.ssh/authorized_keys.new ~/.ssh/authorized_keys

  exit $${STATUS}
EOT
  }
}
