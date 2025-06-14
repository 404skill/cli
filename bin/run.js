#!/usr/bin/env node
const {spawnSync} = require('node:child_process');
const {join} = require('node:path');

const MAP = {
    'darwin-arm64' : 'cli_darwin_arm64_v8.0/404skill',
    'darwin-x64'   : 'cli_darwin_amd64_v1/404skill',
    'linux-arm64'  : 'cli_linux_arm64_v8.0/404skill',
    'linux-x64'    : 'cli_linux_amd64_v1/404skill',
    'win32-x64'    : 'cli_windows_amd64_v1/404skill.exe'
};

const key = `${process.platform}-${process.arch}`;
const bin = MAP[key] && join(__dirname, '..', 'dist', MAP[key]);
if (!bin) {
  console.error(`Unsupported platform: ${key}`);
  process.exit(1);
}

const {status} = spawnSync(bin, process.argv.slice(2), {stdio: 'inherit'});
process.exit(status ?? 0);
