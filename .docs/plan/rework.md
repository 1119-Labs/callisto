```mermaid
flowchart LR
  A[Enqueuer missing + new blocks] -->|publish block_height| Q1[(RabbitMQ block-queue)]
  Q1 --> BW[Block Worker]
  BW -->|RPC get block + tx list| RPC[(Node RPC)]
  BW --> DB1[(DB blocks and consensus)]
  BW -->|publish tx hash x N| Q2[(RabbitMQ tx-queue)]
  Q2 --> TW[Tx Worker]
  TW -->|API gRPC get tx details| API[(Node API gRPC)]
  TW --> DB2[(DB txs messages modules)]
  Q1 --> DLQ1[(DLQ block-queue)]
  Q2 --> DLQ2[(DLQ tx-queue)]
  ```