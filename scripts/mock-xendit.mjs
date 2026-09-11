import { createServer } from "node:http";

const server = createServer((request, response) => {
  if (request.method === "POST" && request.url === "/v2/invoices") {
    let raw = "";
    request.on("data", (chunk) => { raw += chunk; });
    request.on("end", () => {
      const payload = JSON.parse(raw || "{}");
      response.writeHead(200, { "Content-Type": "application/json" });
      response.end(JSON.stringify({
        id: `mock-invoice-${Date.now()}`,
        external_id: payload.external_id,
        status: "PENDING",
        amount: payload.amount,
        invoice_url: "http://localhost:9090/mock-payment",
        expiry_date: new Date(Date.now() + 3600000).toISOString(),
      }));
    });
    return;
  }

  response.writeHead(404, { "Content-Type": "application/json" });
  response.end(JSON.stringify({ message: "not found" }));
});

server.listen(9090, "127.0.0.1", () => {
  console.log("mock Xendit listening on http://127.0.0.1:9090");
});
