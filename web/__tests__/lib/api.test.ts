import { chat } from "@/lib/api";

// Mock global fetch
const mockFetch = jest.fn();
global.fetch = mockFetch;

describe("chat API client", () => {
  beforeEach(() => {
    mockFetch.mockClear();
  });

  it("sends POST request with question", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ answer: "The fuse is here", sources: [] }),
    });

    const result = await chat("Where is the fuse?");
    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/api/chat",
      expect.objectContaining({
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ question: "Where is the fuse?", vehicle_model: undefined }),
      })
    );
    expect(result.answer).toBe("The fuse is here");
  });

  it("includes vehicle_model when provided", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ answer: "Answer", sources: [] }),
    });

    await chat("Question", "Honda Civic 2020");
    expect(mockFetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        body: JSON.stringify({ question: "Question", vehicle_model: "Honda Civic 2020" }),
      })
    );
  });

  it("throws on non-ok response", async () => {
    mockFetch.mockResolvedValue({ ok: false, status: 500 });
    await expect(chat("test")).rejects.toThrow("API error: 500");
  });

  it("throws on network failure", async () => {
    mockFetch.mockRejectedValue(new Error("Network error"));
    await expect(chat("test")).rejects.toThrow("Network error");
  });

  it("returns sources from response", async () => {
    const sources = [
      { title: "Manual", source_type: "pdf", score: 0.9 },
    ];
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ answer: "Answer", sources }),
    });

    const result = await chat("test");
    expect(result.sources).toEqual(sources);
  });
});
