from typing import List, Optional

class Cluster:
    def __init__(self, 
                 name: str, 
                 servers: List[str], 
                 agents: List[str], 
                 port: int, 
                 api_port: int, 
                 ip: str = None):
        self.name = name
        self.servers = servers
        self.agents = agents
        self.port = port
        self.api_port = api_port
        self.ip = ip  # Add ip attribute

    def to_dict(self):
        return {
            'name': self.name,
            'servers': self.servers,
            'agents': self.agents,
            'port': self.port,
            'api_port': self.api_port,
            'ip': self.ip  # Add ip to dict
        }

    @classmethod
    def from_dict(cls, data: dict):
        return cls(
            name=data['name'],
            servers=data['servers'],
            agents=data['agents'],
            port=data['port'],
            api_port=data['api_port'],
            ip=data.get('ip')  # Get ip from dict
        )