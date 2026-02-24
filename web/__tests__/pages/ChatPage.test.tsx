import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ChatPage from "@/app/chat/page";

// Mock the API module
jest.mock("@/lib/api", () => ({
  chat: jest.fn(),
  chatStream: jest.fn(),
}));

// Mock scrollIntoView
window.HTMLElement.prototype.scrollIntoView = jest.fn();

import { chat, chatStream } from "@/lib/api";

const mockChat = chat as jest.MockedFunction<typeof chat>;
const mockChatStream = chatStream as jest.MockedFunction<typeof chatStream>;

describe("ChatPage", () => {
  beforeEach(() => {
    mockChat.mockClear();
    mockChatStream.mockClear();
    // Default: streaming fails, falls back to chat
    mockChatStream.mockImplementation(async function* () {
      throw new Error("streaming not available");
    });
  });

  it("renders initial state with welcome message", () => {
    render(<ChatPage />);
    expect(screen.getByText("Welcome to Wessley")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Ask about any vehicle...")).toBeInTheDocument();
  });

  it("sends a message and displays response", async () => {
    mockChat.mockResolvedValue({
      answer: "The relay is under the hood.",
      sources: [],
    });

    const user = userEvent.setup();
    render(<ChatPage />);

    const textarea = screen.getByPlaceholderText("Ask about any vehicle...");
    await user.type(textarea, "Where is the relay?");
    await user.click(screen.getByRole("button", { name: "Send" }));

    // User message should appear
    expect(screen.getByText("Where is the relay?")).toBeInTheDocument();

    // Wait for assistant response
    await waitFor(() => {
      expect(screen.getByText("The relay is under the hood.")).toBeInTheDocument();
    });
  });

  it("displays error message on API failure", async () => {
    mockChatStream.mockImplementation(async function* () {
      throw new Error("stream fail");
    });
    mockChat.mockRejectedValue(new Error("API error: 500"));

    const user = userEvent.setup();
    render(<ChatPage />);

    const textarea = screen.getByPlaceholderText("Ask about any vehicle...");
    await user.type(textarea, "test");
    await user.click(screen.getByRole("button", { name: "Send" }));

    await waitFor(() => {
      expect(screen.getByText(/API error: 500/)).toBeInTheDocument();
    });
  });

  it("disables input while loading", async () => {
    // Make chat hang
    mockChat.mockImplementation(() => new Promise(() => {}));

    const user = userEvent.setup();
    render(<ChatPage />);

    const textarea = screen.getByPlaceholderText("Ask about any vehicle...");
    await user.type(textarea, "test");
    await user.click(screen.getByRole("button", { name: "Send" }));

    // Input should be disabled while waiting
    await waitFor(() => {
      expect(screen.getByPlaceholderText("Ask about any vehicle...")).toBeDisabled();
    });
  });
});
