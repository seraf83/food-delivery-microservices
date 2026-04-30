<?php

declare(strict_types=1);

namespace App\Entity;

use Doctrine\ORM\Mapping as ORM;

#[ORM\Entity]
#[ORM\Table(name: 'orders')]
class Order
{
    #[ORM\Id]
    #[ORM\GeneratedValue]
    #[ORM\Column]
    private int $id;

    #[ORM\Column]
    private int $userId;

    #[ORM\Column(length: 500)]
    private string $deliveryAddress;

    #[ORM\Column(type: 'json')]
    private array $items;

    #[ORM\Column(type: 'decimal', precision: 10, scale: 2)]
    private float $totalAmount;

    #[ORM\Column(length: 50)]
    private string $status = 'pending';

    #[ORM\Column(nullable: true)]
    private ?int $courierId = null;

    #[ORM\Column(nullable: true)]
    private ?int $estimatedMinutes = null;

    #[ORM\Column(nullable: true)]
    private ?\DateTime $deliveredAt = null;

    #[ORM\Column]
    private \DateTime $createdAt;

    public function __construct(
        int    $userId,
        string $deliveryAddress,
        array  $items,
        float  $totalAmount,
    )
    {
        $this->userId = $userId;
        $this->deliveryAddress = $deliveryAddress;
        $this->items = $items;
        $this->totalAmount = $totalAmount;
        $this->createdAt = new \DateTime();
    }

    public function getId(): int
    {
        return $this->id;
    }

    public function getUserId(): int
    {
        return $this->userId;
    }

    public function getDeliveryAddress(): string
    {
        return $this->deliveryAddress;
    }

    public function getItems(): array
    {
        return $this->items;
    }

    public function getTotalAmount(): float
    {
        return $this->totalAmount;
    }

    public function getStatus(): string
    {
        return $this->status;
    }

    public function setStatus(string $status): void
    {
        $this->status = $status;
    }

    public function setCourierId(int $id): void
    {
        $this->courierId = $id;
    }

    public function setEstimatedMinutes(int $min): void
    {
        $this->estimatedMinutes = $min;
    }

    public function setDeliveredAt(\DateTime $dt): void
    {
        $this->deliveredAt = $dt;
    }

    public function toArray(): array
    {
        return [
            'id' => $this->id,
            'user_id' => $this->userId,
            'address' => $this->deliveryAddress,
            'items' => $this->items,
            'total' => $this->totalAmount,
            'status' => $this->status,
            'courier_id' => $this->courierId,
            'eta_minutes' => $this->estimatedMinutes,
            'delivered_at' => $this->deliveredAt?->format('Y-m-d H:i:s'),
            'created_at' => $this->createdAt->format('Y-m-d H:i:s'),
        ];
    }
}
