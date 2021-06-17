.PHONY: build clean rebuild
build: wordpress/.unarchived
rebuild: clean build
clean:
	rm -rf wordpress-11.0.4.tgz wordpress

wordpress-11.0.4.tgz:
	helm repo add bitnami https://charts.bitnami.com/bitnami
	helm pull bitnami/wordpress --version 11.0.4

wordpress/.unarchived: wordpress-11.0.4.tgz
	tar zxvf wordpress-11.0.4.tgz
	touch wordpress/.unarchived

