#!/bin/sh
set -e

systemctl stop wp-task-runner.service || true
systemctl disable wp-task-runner.service || true
