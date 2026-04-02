# API Reference

## Auth

| Method | Path              | Auth | Description   |
| ------ | ----------------- | ---- | ------------- |
| POST   | /api/auth/login   | -    | Log in        |
| POST   | /api/auth/refresh | -    | Refresh token |
| POST   | /api/auth/logout  | -    | Log out       |
| GET    | /api/user/me      | JWT  | Current user  |

## Projects & Environments

| Method | Path                           | Auth | Description      |
| ------ | ------------------------------ | ---- | ---------------- |
| GET    | /api/projects                  | JWT  | List projects    |
| POST   | /api/projects                  | JWT  | Create project   |
| GET    | /api/projects/:id              | JWT  | Get project      |
| PUT    | /api/projects/:id              | JWT  | Update project   |
| DELETE | /api/projects/:id              | JWT  | Delete project   |
| POST   | /api/projects/:id/environments | JWT  | Add environment  |
| POST   | /api/environments/:envId/fork  | JWT  | Fork environment |

## Resources

| Method | Path                                        | Auth | Description                  |
| ------ | ------------------------------------------- | ---- | ---------------------------- |
| GET    | /api/environments/:envId/resources          | JWT  | List resources in env        |
| POST   | /api/environments/:envId/resources          | JWT  | Create resource (from image) |
| POST   | /api/environments/:envId/resources/from-git | JWT  | Create resource from git     |
| GET    | /api/resources/:id                          | JWT  | Get resource                 |
| PUT    | /api/resources/:id                          | JWT  | Update resource              |
| DELETE | /api/resources/:id                          | JWT  | Delete resource              |
| POST   | /api/resources/:id/start                    | JWT  | Start resource               |
| POST   | /api/resources/:id/stop                     | JWT  | Stop resource                |
| POST   | /api/resources/:id/build                    | JWT  | Trigger build                |
| GET    | /api/resources/:id/logs                     | JWT  | Get run logs                 |
| GET    | /api/resources/:id/env-vars                 | JWT  | List env vars                |
| PUT    | /api/resources/:id/env-vars                 | JWT  | Update env vars              |

## Database Backups

| Method | Path                                  | Auth | Description                 |
| ------ | ------------------------------------- | ---- | --------------------------- |
| POST   | /api/backup-sources                   | JWT  | Create backup source        |
| GET    | /api/backup-sources                   | JWT  | List backup sources         |
| GET    | /api/backup-sources/:id               | JWT  | Get backup source           |
| PUT    | /api/backup-sources/:id               | JWT  | Update backup source        |
| DELETE | /api/backup-sources/:id               | JWT  | Delete backup source        |
| POST   | /api/storages                         | JWT  | Create backup storage       |
| GET    | /api/storages                         | JWT  | List backup storages        |
| GET    | /api/storages/:id                     | JWT  | Get backup storage          |
| PUT    | /api/storages/:id                     | JWT  | Update backup storage       |
| DELETE | /api/storages/:id                     | JWT  | Delete backup storage       |
| POST   | /api/backup-configs                   | JWT  | Create backup config        |
| GET    | /api/backup-configs/:id               | JWT  | Get backup config           |
| GET    | /api/backup-sources/:id/backup-config | JWT  | Get backup config by source |
| PUT    | /api/backup-configs/:id               | JWT  | Update backup config        |
| POST   | /api/backup-sources/:id/backups       | JWT  | Trigger backup              |
| GET    | /api/backup-sources/:id/backups       | JWT  | List backups for one source |
| GET    | /api/backups/:id                      | JWT  | Get backup                  |
| POST   | /api/backups/:id/restore              | JWT  | Trigger restore             |
| GET    | /api/restores/:id                     | JWT  | Get restore                 |

## Routing & Settings

| Method | Path                           | Auth | Description                        |
| ------ | ------------------------------ | ---- | ---------------------------------- |
| GET    | /api/settings                  | JWT  | Get platform settings              |
| PATCH  | /api/settings                  | JWT  | Update platform settings           |
| GET    | /api/settings/base-domains     | JWT  | List managed base domains          |
| POST   | /api/settings/base-domains     | JWT  | Add base domain                    |
| DELETE | /api/settings/base-domains/:id | JWT  | Delete base domain                 |
| GET    | /api/domains/check             | JWT  | Check whether a hostname is in use |

## Builds

| Method | Path                   | Auth | Description               |
| ------ | ---------------------- | ---- | ------------------------- |
| GET    | /api/builds            | JWT  | List build jobs           |
| POST   | /api/builds            | JWT  | Create build from git     |
| POST   | /api/builds/upload     | JWT  | Build from archive upload |
| GET    | /api/builds/:id        | JWT  | Get build job             |
| POST   | /api/builds/:id/cancel | JWT  | Cancel build              |

## Source Connections (GitHub)

| Method | Path                                                    | Auth | Description             |
| ------ | ------------------------------------------------------- | ---- | ----------------------- |
| POST   | /api/source-connections/github/apps                     | JWT  | Begin GitHub OAuth flow |
| POST   | /api/source-connections/pat                             | JWT  | Add PAT connection      |
| GET    | /api/source-connections                                 | JWT  | List connections        |
| DELETE | /api/source-connections/:id                             | JWT  | Remove connection       |
| GET    | /api/source-connections/:id/repos                       | JWT  | List repos              |
| GET    | /api/source-connections/:id/repos/:owner/:repo/branches | JWT  | List branches           |

## Channels

| Method | Path              | Auth | Description    |
| ------ | ----------------- | ---- | -------------- |
| GET    | /api/channels     | JWT  | List channels  |
| POST   | /api/channels     | JWT  | Create channel |
| GET    | /api/channels/:id | JWT  | Get channel    |
| DELETE | /api/channels/:id | JWT  | Delete channel |

## WebSocket

| Path                           | Description                 |
| ------------------------------ | --------------------------- |
| /api/ws/builds/:id             | Stream build logs           |
| /api/ws/resource-runs/:id      | Stream resource run logs    |
| /api/ws/resources/:id/terminal | Interactive container shell |
