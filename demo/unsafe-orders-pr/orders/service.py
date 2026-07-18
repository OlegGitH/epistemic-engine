import logging

logger = logging.getLogger("orders")


def process_order(order_id: str, customer_email: str) -> None:
    # Intentional defect: this exposes customer PII to application logs.
    logger.info("processing order=%s customer_email=%s", order_id, customer_email)
