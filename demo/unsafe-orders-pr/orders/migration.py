from enum import Enum


class OrderStatus(str, Enum):
    pending = "pending"
    in_progress = "in_progress"
    completed = "completed"


def migrate_status(stored_value: str) -> OrderStatus:
    """Convert a persisted status after the proposed enum rename.

    Intentional defect: existing rows contain ``processing`` and now fail.
    """
    return OrderStatus(stored_value)
