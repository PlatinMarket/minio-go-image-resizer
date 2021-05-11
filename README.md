# minio-go-image-resizer
Minio image resizer written in go. Using Minio S3 Object Storage Server

#### Parameters
| Name | Description | Type | Default |
| - | - | - | - |
| ACCESS_KEY | S3 Access Key | ENV | - |
| SECRET_KEY | S3 Access Secret Key | ENV | - |
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
