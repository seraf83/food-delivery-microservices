<?php

declare(strict_types=1);

namespace App\Entity;

use Doctrine\ORM\Mapping as ORM;

#[ORM\Entity]
#[ORM\Table(name: 'couriers')]
class Courier
{
    #[ORM\Id]
    #[ORM\GeneratedValue]
    #[ORM\Column]
    private int $id;

    #[ORM\Column(length: 100)]
    private string $name;

    #[ORM\Column(length: 20)]
    private string $phone;

    #[ORM\Column(length: 20)]
    private string $status = 'available';

    public function __construct(string $name, string $phone)
    {
        $this->name  = $name;
        $this->phone = $phone;
    }

    public function getId(): int        { return $this->id; }
    public function getName(): string   { return $this->name; }
    public function getPhone(): string  { return $this->phone; }
    public function getStatus(): string { return $this->status; }

    public function setStatus(string $status): void { $this->status = $status; }
}
