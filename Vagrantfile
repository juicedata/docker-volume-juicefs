$script = <<-SCRIPT
curl -s https://get.docker.com |sudo sh
sudo usermod -aG docker vagrant
SCRIPT

Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/xenial64"
  config.vm.provision "shell", inline: $script
end
