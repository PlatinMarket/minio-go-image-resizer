# minio-go-image-resizer
Minio image resizer written in go. Using Mino S3 Object Storage Server

#### Parameters
| Name | Description | Type | Default |
| - | - | - | - |
| ACCESS_KEY | S3 Access Key | ENV | - |
| SECRET_KEY | S3 Access Secret Key | ENV | - |
| -b | Bucket Name | PARAMETER | - |
| -a | Server Listen Address | PARAMETER | "0.0.0.0:2222" |
| -e | Minio Address | PARAMETER | - |

#### Usage

```bash
resizer -b reform -a "0.0.0.0:2222" -e "http://localhost:9000"
```