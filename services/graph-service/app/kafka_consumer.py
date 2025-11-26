"""
Kafka Consumer - Listens for paper.processed events and builds graph
"""

import asyncio
import json
import logging
from typing import Optional
from aiokafka import AIOKafkaConsumer
from aiokafka.errors import KafkaError

logger = logging.getLogger(__name__)


class GraphKafkaConsumer:
    """Consumes paper processing events from Kafka and builds graph"""

    def __init__(
        self,
        bootstrap_servers: str,
        topic: str,
        group_id: str,
        worker_queue
    ):
        self.bootstrap_servers = bootstrap_servers
        self.topic = topic
        self.group_id = group_id
        self.worker_queue = worker_queue
        self.consumer: Optional[AIOKafkaConsumer] = None
        self.is_running = False

    async def start(self):
        """Start Kafka consumer with retry logic"""
        max_retries = 10
        retry_delay = 3  # seconds

        for attempt in range(max_retries):
            try:
                logger.info(f"🔌 Connecting to Kafka at {self.bootstrap_servers} (attempt {attempt + 1}/{max_retries})...")

                self.consumer = AIOKafkaConsumer(
                    self.topic,
                    bootstrap_servers=self.bootstrap_servers,
                    group_id=self.group_id,
                    value_deserializer=lambda m: json.loads(m.decode('utf-8')),
                    auto_offset_reset='earliest',  # Start from beginning if no offset
                    enable_auto_commit=True,
                    auto_commit_interval_ms=1000
                )

                await self.consumer.start()
                self.is_running = True

                logger.info(f"✅ Kafka consumer started. Listening to topic: {self.topic}")

                # Start consuming messages
                asyncio.create_task(self._consume_messages())
                return  # Success!

            except Exception as e:
                logger.warning(f"⚠️  Kafka connection attempt {attempt + 1} failed: {e}")
                if attempt < max_retries - 1:
                    logger.info(f"⏳ Retrying in {retry_delay} seconds...")
                    await asyncio.sleep(retry_delay)
                else:
                    logger.error(f"❌ Failed to connect to Kafka after {max_retries} attempts")
                    raise

    async def _consume_messages(self):
        """Consume messages from Kafka"""
        logger.info("📥 Started consuming messages from Kafka...")

        try:
            async for message in self.consumer:
                try:
                    # Parse message
                    paper_data = message.value

                    logger.info(f"📨 Received message: {paper_data.get('paper_title', 'Unknown')}")

                    # Validate message
                    if not self._validate_message(paper_data):
                        logger.warning(f"⚠️  Invalid message format: {message.value}")
                        continue

                    # Submit to worker queue for processing
                    await self.worker_queue.submit_job(
                        paper_title=paper_data['paper_title'],
                        latex_content=paper_data['latex_content'],
                        pdf_path=paper_data['pdf_path'],
                        processed_at=paper_data.get('processed_at'),
                        priority=paper_data.get('priority', 0)
                    )

                    logger.info(f"✅ Queued for graph building: {paper_data['paper_title']}")

                except Exception as e:
                    logger.error(f"❌ Error processing message: {e}")
                    # Continue processing next messages
                    continue

        except Exception as e:
            logger.error(f"❌ Consumer error: {e}")
        finally:
            logger.info("👋 Stopped consuming messages")

    def _validate_message(self, data: dict) -> bool:
        """Validate message has required fields"""
        required_fields = ['paper_title', 'latex_content', 'pdf_path']

        for field in required_fields:
            if field not in data or not data[field]:
                return False

        return True

    async def stop(self):
        """Stop Kafka consumer"""
        logger.info("🛑 Stopping Kafka consumer...")

        self.is_running = False

        if self.consumer:
            await self.consumer.stop()

        logger.info("✅ Kafka consumer stopped")

    def get_stats(self) -> dict:
        """Get consumer statistics"""
        return {
            "is_running": self.is_running,
            "topic": self.topic,
            "group_id": self.group_id
        }
