# VireFS

**VireFS** is a lightweight filesystem abstraction layer for Go.

It provides a unified interface to access different storage backends such as **local filesystems and object storage (e.g. S3)** through a single, consistent API.

The goal of VireFS is to make file operations **backend-agnostic**, allowing applications to switch or combine storage systems without changing business logic.

---

## Features

* Unified filesystem abstraction
* Multiple storage backends
* Simple and idiomatic Go API
* Easy backend switching (local ↔ object storage)
* Designed for cloud-native applications
* Extensible driver architecture
