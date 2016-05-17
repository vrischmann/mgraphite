Vagrant.configure(2) do |config|
  config.vm.box = "bento/debian-8.2"

  config.vm.network "forwarded_port", guest: 2003, host: 2003
  config.vm.provision "shell", path: "vagrant_provision.sh"
end
