# Inforo Core Module

The core module of the Inforo system providing essential functionality for component management, task execution, and monitoring.

## Overview

The `core` package is the central module that orchestrates all major subsystems:
- Component management
- Task execution
- Monitoring
- Planning

## Installation

```bash
go get github.com/laplasd/inforo
```

## Core Structure
The main `Core` struct contains registries for different subsystems:

```go
type Core struct {
    logger             *logrus.Logger
    Components         api.ComponentRegistry
    Controllers        api.ControllerRegistry
    Monitorings        api.MonitoringRegistry
    MonitorControllers api.MonitoringControllerRegistry
    Tasks              api.TaskRegistry
    Plans              api.PlanRegistry
}
```

## Initialization
### Default Core

```go
core := NewDefaultCore()
```