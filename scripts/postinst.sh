#!/bin/sh
set -e

# Create log file owned by the ubuntu user
touch /var/log/wp-task-runner.log
chown ubuntu:ubuntu /var/log/wp-task-runner.log
chmod 644 /var/log/wp-task-runner.log

systemctl daemon-reload
systemctl enable --now wp-task-runner.service || true
