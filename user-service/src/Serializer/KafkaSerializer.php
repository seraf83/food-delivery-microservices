<?php

declare(strict_types=1);

namespace App\Serializer;

use Symfony\Component\Messenger\Envelope;
use Symfony\Component\Messenger\Stamp\RedeliveryStamp;
use Symfony\Component\Messenger\Transport\Serialization\SerializerInterface;

class KafkaSerializer implements SerializerInterface
{
    // Messenger counts retries via RedeliveryStamp; without carrying it through
    // Kafka every retry looks like the first one and is retried forever.
    private const RETRY_HEADER = 'X-Retry-Count';

    public function decode(array $encodedEnvelope): Envelope
    {
        $bodyRaw = $encodedEnvelope['body'] ?? null;

        if (!$bodyRaw) {
            throw new \RuntimeException('Empty message body, skipping');
        }

        $body = json_decode($bodyRaw, true);

        if (!isset($body['type'], $body['data'])) {
            throw new \RuntimeException('Invalid message format, skipping');
        }

        $class = $body['type'];
        $message = new $class(...array_values($body['data']));

        $stamps = [];
        $retryCount = $encodedEnvelope['headers'][self::RETRY_HEADER] ?? null;
        if ($retryCount !== null) {
            $stamps[] = new RedeliveryStamp((int) $retryCount);
        }

        return new Envelope($message, $stamps);
    }

    public function encode(Envelope $envelope): array
    {
        $message = $envelope->getMessage();
        $reflection = new \ReflectionClass($message);
        $data = [];
        foreach ($reflection->getProperties() as $property) {
            $data[$property->getName()] = $property->getValue($message);
        }

        $headers = ['Content-Type' => 'application/json'];
        $redelivery = $envelope->last(RedeliveryStamp::class);
        if ($redelivery instanceof RedeliveryStamp) {
            $headers[self::RETRY_HEADER] = (string) $redelivery->getRetryCount();
        }

        return [
            'body' => json_encode([
                'type' => get_class($message),
                'data' => $data,
            ]),
            'headers' => $headers,
        ];
    }
}
