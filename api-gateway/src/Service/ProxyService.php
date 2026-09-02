<?php

namespace App\Service;

use Symfony\Component\HttpFoundation\Request;
use Symfony\Component\HttpFoundation\Response;
use Symfony\Contracts\HttpClient\HttpClientInterface;

class ProxyService
{

    public function __construct(private readonly HttpClientInterface $httpClient)
    {

    }

    public function proxy(string $url, Request $request, array $extraHeaders = []): Response
    {
        $headers = array_merge(
            ['Content-Type' => 'application/json'],
            $extraHeaders,
        );

        $response = $this->httpClient->request(
            method: $request->getMethod(),
            url: $url,
            options: [
                'headers' => $headers,
                'body' => $request->getContent(),

            ],
        );

        return new Response(
            content: $response->getContent(throw: false),
            status: $response->getStatusCode(),
            headers: ['Content-Type' => 'application/json'],
        );
    }
}
