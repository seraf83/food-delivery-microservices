<?php

declare(strict_types=1);

namespace App\Entity;

use Doctrine\ORM\Mapping as ORM;

#[ORM\Entity]
#[ORM\Table(name: 'notifications')]
class Notification
{
    #[ORM\Id]
    #[ORM\GeneratedValue]
    #[ORM\Column]
    private int $id;

    #[ORM\Column]
    private int $userId;

    #[ORM\Column(length: 500)]
    private string $message;

    #[ORM\Column(length: 50)]
    private string $type;

    #[ORM\Column]
    private \DateTime $createdAt;

    public function __construct(
        int    $userId,
        string $message,
        string $type,
    )
    {
        $this->userId = $userId;
        $this->message = $message;
        $this->type = $type;
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

    public function getMessage(): string
    {
        return $this->message;
    }

    public function getType(): string
    {
        return $this->type;
    }
}
