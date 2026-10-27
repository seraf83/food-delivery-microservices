<?php

declare(strict_types=1);

namespace App\Serializer;

use Symfony\Component\Messenger\Envelope;
use Symfony\Component\Messenger\Transport\Serialization\SerializerInterface;

class KafkaSerializer implements SerializerInterface
{
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
        return new Envelope($message);
    }

    public function encode(Envelope $envelope): array
    {
        $message = $envelope->getMessage();
        $reflection = new \ReflectionClass($message);
        $data = [];
        foreach ($reflection->getProperties() as $property) {
            $data[$property->getName()] = $property->getValue($message);
        }

        return [
            'body' => json_encode([
                'type' => get_class($message),
                'data' => $data,
            ]),
            'headers' => ['Content-Type' => 'application/json'],
        ];
    }
}
