import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ChatInput from "@/components/ChatInput";

describe("ChatInput", () => {
  const mockSend = jest.fn();

  beforeEach(() => {
    mockSend.mockClear();
  });

  it("renders textarea and send button", () => {
    render(<ChatInput onSend={mockSend} disabled={false} />);
    expect(screen.getByPlaceholderText("Ask about any vehicle...")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Send" })).toBeInTheDocument();
  });

  it("calls onSend with trimmed input on submit", async () => {
    const user = userEvent.setup();
    render(<ChatInput onSend={mockSend} disabled={false} />);
    const textarea = screen.getByPlaceholderText("Ask about any vehicle...");
    await user.type(textarea, "hello world");
    await user.click(screen.getByRole("button", { name: "Send" }));
    expect(mockSend).toHaveBeenCalledWith("hello world");
  });

  it("does not call onSend when input is empty", async () => {
    const user = userEvent.setup();
    render(<ChatInput onSend={mockSend} disabled={false} />);
    await user.click(screen.getByRole("button", { name: "Send" }));
    expect(mockSend).not.toHaveBeenCalled();
  });

  it("does not call onSend when input is only whitespace", async () => {
    const user = userEvent.setup();
    render(<ChatInput onSend={mockSend} disabled={false} />);
    const textarea = screen.getByPlaceholderText("Ask about any vehicle...");
    await user.type(textarea, "   ");
    await user.click(screen.getByRole("button", { name: "Send" }));
    expect(mockSend).not.toHaveBeenCalled();
  });

  it("disables textarea and button when disabled prop is true", () => {
    render(<ChatInput onSend={mockSend} disabled={true} />);
    expect(screen.getByPlaceholderText("Ask about any vehicle...")).toBeDisabled();
  });

  it("clears input after successful submit", async () => {
    const user = userEvent.setup();
    render(<ChatInput onSend={mockSend} disabled={false} />);
    const textarea = screen.getByPlaceholderText("Ask about any vehicle...");
    await user.type(textarea, "test");
    await user.click(screen.getByRole("button", { name: "Send" }));
    expect(textarea).toHaveValue("");
  });

  it("shows starter buttons when input is empty and not disabled", () => {
    render(<ChatInput onSend={mockSend} disabled={false} />);
    // There should be starter suggestion buttons
    const buttons = screen.getAllByRole("button");
    // 1 Send button + 3 starter buttons
    expect(buttons.length).toBeGreaterThanOrEqual(4);
  });

  it("calls onSend when a starter button is clicked", async () => {
    const user = userEvent.setup();
    render(<ChatInput onSend={mockSend} disabled={false} />);
    const buttons = screen.getAllByRole("button");
    // Click first starter (not the Send button)
    const starterBtn = buttons.find((b) => b.textContent !== "Send");
    await user.click(starterBtn!);
    expect(mockSend).toHaveBeenCalled();
  });
});
