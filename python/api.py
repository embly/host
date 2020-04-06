from typing import Union, List, Optional, Dict


class Container:
    pass


class Service:
    pass


class LoadBalancer:
    pass


def service(name: str, count: int = 1, containers: List[Container] = []) -> Service:
    pass


def container(
    name: str,
    image: str,
    cpu: int,
    memory: int,
    command: str = "",
    ports: Optional[List[Union[int, str]]] = [],
    connect_to: List[str] = [],
    environment: Dict[str, Union[str, int]] = {},
) -> Container:
    pass


def load_balacer(name: str, routes: Dict[str, str]) -> LoadBalancer:
    pass
