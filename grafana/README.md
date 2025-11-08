# Grafana Configuration

This directory contains Grafana provisioning configuration for the Ephoros project.

## Structure

- `provisioning/dashboards/` - Dashboard provisioning configuration
- `provisioning/dashboards/models/` - Dashboard JSON models
- `provisioning/datasources/` - Datasource provisioning configuration

## Usage

Grafana is automatically configured when running `docker-compose up` using these provisioning files.

The Grafana instance will be available at: http://localhost:3000
