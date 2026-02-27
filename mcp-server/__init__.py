#!/usr/bin/env python3
"""
uiscloud_weknora MCP Server Package

A Model Context Protocol server that provides access to the uiscloud_weknora knowledge management API.
"""

__version__ = "1.0.0"
__author__ = "uiscloud_weknora Team"
__description__ = "uiscloud_weknora MCP Server - Model Context Protocol server for uiscloud_weknora API"

from .weknora_mcp_server import WeKnoraClient, run

__all__ = ["WeKnoraClient", "run"]
