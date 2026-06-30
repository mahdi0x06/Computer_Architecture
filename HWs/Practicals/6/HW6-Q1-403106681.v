module main (
    input Clk, Rst, CPU_Write, CPU_Read, MM_Done,
    input [31:0] CPU_DataIn, CPU_Addr,
    input [127:0] MM_DataIn,
    output CPU_Done, MM_Write, MM_Read,
    output [31:0] CPU_DataOut, MM_Addr,
    output [127:0] MM_DataOut 
);

    wire [3:0] offset = CPU_Addr[3:0];
    wire [5:0] ind = CPU_Addr[9:4];
    wire [21:0] tag = CPU_Addr[31:10];

    reg [127:0] Data_arr [0:63];
    reg [21:0] tag_arr [0:63];
    reg valid_arr [0:63];

    integer i;
    initial begin
        for (i = 0; i < 64; i = i + 1) begin
            valid_arr[i] = 1'b0;
            tag_arr[i] = 22'b0;
            Data_arr[i] = 128'b0;
        end
    end

    wire cache_hit = (valid_arr[ind] === 1'b1) && (tag_arr[ind] == tag);
    wire [127:0] current_data = Data_arr[ind];

    localparam IDLE        = 3'd0;
    localparam READ_REQ    = 3'd1;
    localparam READ_WAIT   = 3'd2;
    localparam ALLOC_REQ   = 3'd3;
    localparam ALLOC_WAIT  = 3'd4;
    localparam WRITE_REQ   = 3'd5;
    localparam WRITE_WAIT  = 3'd6;

    reg [2:0] state;

    wire [127:0] active_data = (state == READ_WAIT && MM_Done === 1'b1) ? MM_DataIn : current_data;

    wire [127:0] updated_block = 
        (offset[3:2] == 2'b00) ? {current_data[127:32], CPU_DataIn} :
        (offset[3:2] == 2'b01) ? {current_data[127:64], CPU_DataIn, current_data[31:0]} :
        (offset[3:2] == 2'b10) ? {current_data[127:96], CPU_DataIn, current_data[63:0]} :
                                 {CPU_DataIn, current_data[95:0]};

    always @(posedge Clk or posedge Rst) begin
        if (Rst === 1'b1) begin
            state <= IDLE;
        end else begin
            case (state)
                IDLE: begin
                    if (CPU_Read === 1'b1 && !cache_hit) begin
                        state <= READ_REQ;
                    end else if (CPU_Write === 1'b1) begin
                        if (cache_hit) state <= WRITE_REQ;
                        else state <= ALLOC_REQ;
                    end
                end
                
                READ_REQ: begin
                    state <= READ_WAIT;
                end
                READ_WAIT: begin
                    if (MM_Done === 1'b1) begin
                        valid_arr[ind] <= 1'b1;
                        tag_arr[ind] <= tag;
                        Data_arr[ind] <= MM_DataIn;
                        state <= IDLE;
                    end
                end
                
                ALLOC_REQ: begin
                    state <= ALLOC_WAIT;
                end
                ALLOC_WAIT: begin
                    if (MM_Done === 1'b1) begin
                        valid_arr[ind] <= 1'b1;
                        tag_arr[ind] <= tag;
                        Data_arr[ind] <= MM_DataIn;
                        state <= WRITE_REQ;
                    end
                end
                
                WRITE_REQ: begin
                    state <= WRITE_WAIT;
                end
                WRITE_WAIT: begin
                    if (MM_Done === 1'b1) begin
                        Data_arr[ind] <= updated_block;
                        state <= IDLE;
                    end
                end
                
                default: state <= IDLE;
            endcase
        end
    end

    assign MM_Addr = {CPU_Addr[31:4], 4'b0000};
    assign MM_DataOut = updated_block;

    assign MM_Read = (state == READ_REQ) || (state == ALLOC_REQ);
    assign MM_Write = (state == WRITE_REQ);

    assign CPU_Done = (state == IDLE && cache_hit && !CPU_Write) || 
                      (state == READ_WAIT && MM_Done === 1'b1) || 
                      (state == WRITE_WAIT && MM_Done === 1'b1);

    wire [31:0] word_0 = active_data[31:0];
    wire [31:0] word_1 = active_data[63:32];
    wire [31:0] word_2 = active_data[95:64];
    wire [31:0] word_3 = active_data[127:96];

    assign CPU_DataOut = (offset[3:2] == 2'b00) ? word_0 :
                         (offset[3:2] == 2'b01) ? word_1 :
                         (offset[3:2] == 2'b10) ? word_2 :
                         word_3;

endmodule

