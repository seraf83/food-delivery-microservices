<?php

declare(strict_types=1);

namespace App\Entity;

use Doctrine\ORM\Mapping as ORM;

#[ORM\Entity]
#[ORM\Table(name: 'deliveries')]
class Delivery
{
    #[ORM\Id]
    #[ORM\GeneratedValue]
    #[ORM\Column]
    private int $id;

    #[ORM\Column]
    private int $orderId;

    #[ORM\Column]
    private int $userId;

    #[ORM\Column]
    private int $courierId;

    #[ORM\Column(length: 500)]
    private string $address;

    #[ORM\Column(length: 50)]
    private string $status = 'assigned';

    #[ORM\Column(nullable: true)]
    private ?\DateTime $completedAt = null;

    #[ORM\Column]
    private \DateTime $createdAt;

    public function __construct(
        int $orderId,
        int $userId,
        int $courierId,
        string $address,
    ) {
        $this->orderId   = $orderId;
        $this->userId    = $userId;
        $this->courierId = $courierId;
        $this->address   = $address;
        $this->createdAt = new \DateTime();
    }

    public function getId(): int        { return $this->id; }
    public function getOrderId(): int   { return $this->orderId; }
    public function getUserId(): int    { return $this->userId; }
    public function getCourierId(): int { return $this->courierId; }
    public function getStatus(): string { return $this->status; }

    public function setStatus(string $status): void       { $this->status = $status; }
    public function setCompletedAt(\DateTime $dt): void   { $this->completedAt = $dt; }

    public function toArray(): array
    {
        return [
            'id'           => $this->id,
            'order_id'     => $this->orderId,
            'courier_id'   => $this->courierId,
            'status'       => $this->status,
            'completed_at' => $this->completedAt?->format('Y-m-d H:i:s'),
            'created_at'   => $this->createdAt->format('Y-m-d H:i:s'),
        ];
    }
}
