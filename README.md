# AwsEcrHandler

Simple golang application to manage AWS ECR repository images in golang concurrency.

## Functionalities

[Purge Ecr Repository Images](cmd\purgeimage)

- Purge all images in provided via a plain text file, i.e [repositories.txt](repositories.txt)

### Supported Flags

| Flag | Mandatory | default | Description                                |
|------|-----------|--------|--------------------------------------------|
| -f   | T         | ""     | file path containing list of repositories  |
| -r   |           | 50     | channel buffer size on managing repository |
| -i   |           | 100    | channel buffer size on managing image      |
| -rw  |           | 2      | number of worker for image      |
| -iw  |           | 10     | number of worker for image      |


``
go run cmd/purgeimage/* -f ./[repositories.txt](repositories.txt)
``

### Note
AWS cli has rate limit