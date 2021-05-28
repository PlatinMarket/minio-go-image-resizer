# minio-go-image-resizer
Minio image resizer written in go. Using Minio S3 Object Storage Server

#### Parameters
| Name | Description | Type | Default |
| - | - | - | - |
| ACCESS_KEY | S3 Access Key | ENV | - |
| SECRET_KEY | S3 Access Secret Key | ENV | - |
| REDIS_SERVICE | Redis service address | ENV | - |
| -b | Bucket Name | PARAMETER | - |
| -a | Server Listen Address | PARAMETER | "0.0.0.0:2222" |
| -e | Endpoint Address | PARAMETER | "http://minio1.servers.platinbox.org:9000" |
| -r | Region | PARAMETER | "fr-par" |

#### Usage

```bash
$ make build
```

```bash
resizer -b reform -a "0.0.0.0:2222" -e "http://localhost:9000"
```

```bash
ACCESS_KEY=ACCESS SECRET_KEY=SECRET ./bin/resizer -b platinmarket-reform -a 0.0.0.0:2222 -e https://s3.fr-par.scw.cloud
```

```bash
$ docker run -d -p 3333:2222 resizer:latest
$ echo -e "GET /1535/pictures/thumb/100X-AAILHXZZDR32320216321_32819.jpg HTTP/1.0\n\n" | nc 0.0.0.0 3333
```

### Bare Ubuntu Instance

Fill the empty fields on the *thumbnail.env*

```sh
# Install.
$ make install

# Uninstall.
$ sudo make uninstall

# Update.
$ sudo make uninstall && make install
```

```sh
# Start the service.
$ systemctl start resizer.service && systemctl enable resizer.service
```
