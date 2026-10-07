# deploy

> [!NOTE]
> **Disclaimer:** Questa codebase è stata interamente generata tramite strumenti AI, ma è il risultato di una **profonda e meticolosa progettazione architetturale** guidata da un essere umano, curata nei minimi dettagli per garantire robustezza e flessibilità.

> A config-driven deployment automation tool written in Go. Define any workflow once in a JSON file, run it on your laptop as a controller or directly on the server as a standalone agent.

---

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Commands](#core-commands)
- [Concepts](#concepts)
  - [Controller Mode](#controller-mode)
  - [Standalone Mode](#standalone-mode)
  - [Multi-Environment Strategy](#multi-environment-strategy)
  - [No-Trace Mode](#no-trace-mode)
- [Configuration Reference](#configuration-reference)
  - [Root Fields](#root-fields)
  - [`remote`](#remote)
  - [`databases`](#databases)
  - [`actions`](#actions)
- [Step Types](#step-types)
  - [`exec`](#exec)
  - [`upload`](#upload)
  - [`download`](#download)
  - [`copy`](#copy)
  - [`db-backup`](#db-backup)
  - [`db-restore`](#db-restore)
  - [`docker-compose`](#docker-compose)
- [Template Variables](#template-variables)
- [Authentication](#authentication)
- [Pre-flight Checks](#pre-flight-checks)
- [Database Backup & Restore](#database-backup--restore)
- [Deploy History](#deploy-history)
- [CLI Reference](#cli-reference)
- [Windows Compatibility](#windows-compatibility)
- [Examples](#examples)

---

## Overview

`deploy` is a config-driven automation tool for running multi-step workflows — deployments, rollbacks, database snapshots, or anything else you'd normally script manually.

It is **not opinionated** about what your actions are called or what they do. There is no built-in concept of "deploy" or "revert" — those are just action names you define in your config. The tool provides the plumbing: sequential step execution, SSH connectivity, database backup/restore primitives, history tracking, and pre-flight safety checks.

Key features:

- **Config-driven**: the entire workflow lives in a versioned JSON file
- **Dual-mode**: controller (SSH to remote) or standalone (runs locally on the server)
- **History & rollback**: every run is recorded with its commit hash and DB backup paths
- **Database-aware**: automatic snapshots before destructive operations, deterministic restore
- **Multi-database**: Postgres, MySQL, MongoDB, Redis — multiple sources in one config
- **Pre-flight checks**: verifies tool availability and DB connectivity before starting
- **Cross-platform**: runs natively on Windows, Linux, and macOS

