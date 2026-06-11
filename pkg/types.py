from typing import List

class ClusterCreateConfig:
    # ... existing code ...

    def to_k3s_args(self) -> List[str]:
        args = []
        # ... existing code ...

        if self.tls_san:
            args.extend(['--tls-san', self.tls_san])

        # ... existing code ...
        return args