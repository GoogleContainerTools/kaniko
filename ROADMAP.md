# Kaniko Project Roadmap 2024

## Introduction

Kaniko is an open-source project designed to build container images from a Dockerfile, efficiently and securely, in environments that cannot run a Docker daemon. This roadmap outlines our strategic goals and key areas of development for 2024, aligning with our mission to enhance container building in cloud-native environments.

## Vision

- **To be the leading tool for building container images via Dockerfile in cloud-native environments - prioritizing security, efficiency, compatibility, and portability.**

## Strategic Goals

1. **Enhanced Security**: Strengthen security measures in container image building, addressing vulnerabilities and integrating best practices.
2. **Performance Optimization**: Improve build performance to handle large-scale and complex applications.
3. **Ecosystem Compatibility**: Ensure compatibility and integration with a wide range of cloud-native tools and platforms.
4. **Community Engagement**: Foster an active community, encouraging contributions, feedback, and collaboration.

## Key Initiatives

### Q1 & Q2 2024

- **Security Automation Improvements**
  - Add automated image vulnerability scanning and notifications
- **Release Automation Improvements**
  - Improve automation of Kaniko releases to ensure frequent releases with minimal overhead.
- **CI/CD Integration Improvements**
  - Improve integration with popular CI/CD tools (e.g. GitLab CI, Jenkins, GHA, etc.).

### Q3 & Q4 2024

- **Improve Docker Compatibility**
  - Improve kaniko compatibility with docker related to edge cases where current Dockerfiles and resulting images can differ than what docker supports/generates from the same Dockerfile
- **Performance Benchmarking**
  - Implement a performance benchmarking system with performance test suite.
  - Identify performance bottlenecks.
- **Improve Layer Caching Mechanisms**
  - Improve layer caching generally for users (eg: reduce incorrect cache misses and usage).
- **Enhanced Documentation**
  - Update documentation to reflect 2024 best practices and usage patterns.
