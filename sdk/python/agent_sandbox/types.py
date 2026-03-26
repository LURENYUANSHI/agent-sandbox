"""Data models for AgentSandbox API responses."""

from dataclasses import dataclass, field
from typing import Optional, Dict, List, Any


@dataclass
class Sandbox:
    id: str
    name: str
    status: str
    root_dir: str = ""
    created_at: str = ""


@dataclass
class Action:
    id: str = ""
    type: str = ""
    resource: str = ""
    params: Dict[str, str] = field(default_factory=dict)
    metadata: Dict[str, Any] = field(default_factory=dict)
    timestamp: str = ""


@dataclass
class ActionResult:
    action_id: str = ""
    success: bool = False
    output: str = ""
    error: str = ""
    exit_code: int = 0
    duration: int = 0
    bytes_read: int = 0
    bytes_written: int = 0


@dataclass
class PolicyDecision:
    effect: str = ""
    allowed: bool = False
    rule: str = ""
    reason: str = ""


@dataclass
class TraceEvent:
    id: str = ""
    sandbox_id: str = ""
    parent_id: str = ""
    type: str = ""
    action: Optional[Action] = None
    action_id: str = ""
    result: Optional[ActionResult] = None
    policy_decision: Optional[PolicyDecision] = None
    timestamp: str = ""
    duration: int = 0
    data: Dict[str, str] = field(default_factory=dict)
    attributes: Dict[str, str] = field(default_factory=dict)


@dataclass
class Rule:
    id: str = ""
    name: str = ""
    description: str = ""
    actions: List[str] = field(default_factory=list)
    resources: List[str] = field(default_factory=list)
    effect: str = ""
    priority: int = 0
    conditions: Dict[str, str] = field(default_factory=dict)


@dataclass
class Policy:
    name: str = ""
    version: str = ""
    description: str = ""
    default_effect: str = ""
    rules: List[Rule] = field(default_factory=list)
