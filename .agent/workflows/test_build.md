---
description: Test build locally using Podman
---

# Test Build with Podman

To verify the Dockerfile build locally, always use `podman` instead of `docker`.

1. Run the build command:
   ```bash
   podman build . -t test-build
   ```

2. (Optional) Verify the image list:
   ```bash
   podman images | grep test-build
   ```
