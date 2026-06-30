module L1_Cache (
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

module ALU(
    input wire clk,
    input wire [1:0] ALUOp,
    input wire [31:0] Reg1, Reg2,
    output reg [31:0] Result,
    output wire zero,
    output wire ready
);
    wire [31:0] mul_res, div_res;
    wire mul_ready, div_ready;
    reg prev_mul, prev_div;

    initial begin
        prev_mul = 1'b0;
        prev_div = 1'b0;
    end

    always @(posedge clk) begin
        prev_mul <= (ALUOp == 2'b10);
        prev_div <= (ALUOp == 2'b11);
    end

    wire start_mul = (ALUOp == 2'b10) && !prev_mul;
    wire start_div = (ALUOp == 2'b11) && !prev_div;

    MUL mul(clk, start_mul, Reg1, Reg2, mul_res, mul_ready);
    DIV div(clk, start_div, Reg1, Reg2, div_res, div_ready);

    always @(*) begin
        case (ALUOp)
            2'b00: Result = Reg1 + Reg2;
            2'b01: Result = Reg1 - Reg2;
            2'b10: Result = mul_res;
            2'b11: Result = div_res;
            default: Result = 32'b0;
        endcase
    end
    
    assign zero = (Result == 32'b0) ? 1'b1 : 1'b0;
    assign ready = (ALUOp == 2'b10) ? mul_ready :
                   (ALUOp == 2'b11) ? div_ready : 1'b1;
endmodule

module MUL(
    input wire clk,
    input wire start,
    input wire [31:0] Reg1, Reg2,
    output reg [31:0] Result,
    output wire ready
);
    reg [5:0] count;
    reg [31:0] a, b;
    reg busy;
    
    initial busy = 1'b0;
    
    assign ready = ~busy & ~start;

    always @(posedge clk) begin
        if (start) begin
            a <= Reg1;
            b <= Reg2;
            Result <= 32'b0;
            count <= 6'b0;
            busy <= 1'b1;
        end else if (busy) begin
            if (a[0]) Result <= Result + b;
            a <= a >> 1;
            b <= b << 1;
            count <= count + 1;
            if (count == 31) busy <= 1'b0;
        end
    end
endmodule

module DIV(
    input wire clk,
    input wire start,
    input wire [31:0] Reg1, Reg2,
    output reg [31:0] Result,
    output wire ready
);
    reg [5:0] count;
    reg [31:0] q, r, b;
    reg busy;

    initial busy = 1'b0;

    assign ready = ~busy & ~start;

    always @(posedge clk) begin
        if (start) begin
            q <= Reg1;
            b <= Reg2;
            r <= 32'b0;
            Result <= 32'b0;
            count <= 32;
            busy <= 1'b1;
        end else if (busy) begin
            if ({r[30:0], q[31]} >= b) begin
                r <= {r[30:0], q[31]} - b;
                Result[count - 1] <= 1'b1;
            end else begin
                r <= {r[30:0], q[31]};
                Result[count - 1] <= 1'b0;
            end
            q <= q << 1;
            count <= count - 1;
            if (count == 1) busy <= 1'b0;
        end
    end
endmodule

module CU(
    input wire [5:0] opcode, funct,
    output reg [1:0] RegDst, MemtoReg, ALUOp,
    output reg RegWrite, ALUSrc, MemRead, MemWrite, Branch, Jump, Jal, Jr
);
    always @(*) begin
        RegDst = 2'b00;
        MemtoReg = 2'b00;
        ALUOp = 2'b00;
        RegWrite = 1'b0;
        ALUSrc = 1'b0;
        MemRead = 1'b0;
        MemWrite = 1'b0;
        Branch = 1'b0;
        Jump = 1'b0;
        Jal = 1'b0;
        Jr = 1'b0;

        case(opcode) 
            6'b000000:
                //JR
                if(funct == 6'b001000) Jr = 1'b1;
                else begin
                    RegWrite = 1'b1;
                    case(funct)
                        //ADD
                        6'b100000: ALUOp = 2'b00;
                        //SUB
                        6'b100010: ALUOp = 2'b01;
                        //MUL
                        6'b011000: ALUOp = 2'b10;
                        //DIV
                        6'b011010: ALUOp = 2'b11;
                        default: ; 
                    endcase
                end
            
            //LW
            6'b100011: begin ALUSrc = 1'b1; MemtoReg = 2'b01; MemRead = 1'b1; ALUOp = 2'b00; RegWrite = 1'b1; RegDst = 2'b01; end
            // SW
            6'b101011: begin ALUSrc = 1'b1; MemWrite = 1'b1; ALUOp = 2'b00; RegDst = 2'b01; end
            // ADDI
            6'b001000: begin ALUSrc = 1'b1; ALUOp = 2'b00; RegWrite = 1'b1; RegDst = 2'b01; end
            // SUBI
            6'b001001: begin ALUSrc = 1'b1; ALUOp = 2'b01; RegWrite = 1'b1; RegDst = 2'b01; end
            // LUI
            6'b001111: begin ALUSrc = 1'b1; RegWrite = 1'b1; RegDst = 2'b01; MemtoReg = 2'b11; end
            // J
            6'b000010: begin Jump = 1'b1; end
            // JAL
            6'b000011: begin MemtoReg = 2'b10; Jump = 1'b1; Jal = 1'b1; RegWrite = 1'b1; RegDst = 2'b10; end
            // BEQ
            6'b000100: begin Branch = 1'b1; ALUOp = 2'b01; end

            default: ; 
        endcase
    end
endmodule

module RF(
    input wire Clk, Rst, RegWrite,   
    input wire [4:0] ReadReg1, ReadReg2, WriteReg,
    input wire [31:0] Data,
    output wire [31:0] ReadData1, ReadData2,
    output wire [31:0] R0, R1, R2, R3, R4, R5, R6,
    R7, R8, R9, R10, R11, R12, R13, R14, R15,
    R16, R17, R18, R19, R20, R21, R22, R23, 
    R24, R25, R26, R27, R28, R29, R30, R31
);
    reg [31:0] Registers [0:31]; 
    integer i;

    initial begin
        for (i = 0; i < 32; i = i + 1) begin
            Registers[i] = 32'b0;
        end
    end

    assign ReadData1 = (ReadReg1 == 5'd0) ? 32'b0 : Registers[ReadReg1];
    assign ReadData2 = (ReadReg2 == 5'd0) ? 32'b0 : Registers[ReadReg2];
    
    assign {R31, R30, R29, R28, R27, R26, R25, R24, R23, R22, R21, R20, R19, R18, R17, R16, 
        R15, R14, R13, R12, R11, R10, R9, R8, R7, R6, R5, R4, R3, R2, R1, R0} = 
       {Registers[31], Registers[30], Registers[29], Registers[28], Registers[27],
        Registers[26], Registers[25], Registers[24], Registers[23], Registers[22],
        Registers[21], Registers[20], Registers[19], Registers[18], Registers[17],
        Registers[16], Registers[15], Registers[14], Registers[13], Registers[12],
        Registers[11], Registers[10], Registers[9], Registers[8], Registers[7], 
        Registers[6], Registers[5], Registers[4], Registers[3], Registers[2], 
        Registers[1], Registers[0]};

    always @(posedge Clk or posedge Rst) begin
        if (Rst == 1'b1) begin
            for (i = 0; i < 32; i = i + 1) begin
                Registers[i] <= 32'b0;
            end
        end else if (RegWrite == 1'b1 && WriteReg != 5'b00000) begin 
            Registers[WriteReg] <= Data;
        end
    end
endmodule

module main (
    input wire Clk,
    input wire Rst,
    
    input wire [127:0] InstrMM_DataIn,
    input wire InstrMM_Done,
    input wire [127:0] DataMM_DataIn,
    input wire DataMM_Done,
    
    output wire [31:0] PC,
    
    output wire [31:0] InstrMM_Addr,
    output wire InstrMM_Read,
    
    output wire [31:0] DataMM_Addr,
    output wire DataMM_Read,
    output wire DataMM_Write,
    output wire [127:0] DataMM_DataOut,
    
    output wire [31:0] R0, R1, R2, R3, R4, R5, R6, R7, R8, R9, R10, R11, R12, R13, R14, R15,
    output wire [31:0] R16, R17, R18, R19, R20, R21, R22, R23, R24, R25, R26, R27, R28, R29, R30, R31
);

    wire cpu_ready;
    wire inst_cache_done;
    wire data_cache_done;
    wire [31:0] InstrIn;
    wire [31:0] DataIn;

    //PC
    reg [31:0] PC_reg = 32'd2044; 
    wire [31:0] next_pc;
    wire [31:0] pc_plus_4 = PC_reg + 32'd4;
    wire alu_ready;

    assign PC = PC_reg >> 2;

    always @(posedge Clk or posedge Rst) begin
        if (Rst)
            PC_reg <= 32'b0;
        else if (alu_ready & cpu_ready)
            PC_reg <= next_pc;
    end
    //
    
    wire inst_mm_write_nc;
    wire [127:0] inst_mm_dataout_nc;
    
    L1_Cache inst_cache (
        .Clk(Clk),
        .Rst(Rst),
        .CPU_Write(1'b0),
        .CPU_Read(1'b1),
        .MM_Done(InstrMM_Done),
        .CPU_DataIn(32'b0),
        .CPU_Addr(PC_reg),
        .MM_DataIn(InstrMM_DataIn),
        .CPU_Done(inst_cache_done),
        .MM_Write(inst_mm_write_nc),
        .MM_Read(InstrMM_Read),
        .CPU_DataOut(InstrIn),
        .MM_Addr(InstrMM_Addr),
        .MM_DataOut(inst_mm_dataout_nc)
    );

    wire [5:0]  opcode = InstrIn[31:26];
    wire [4:0]  rs     = InstrIn[25:21];
    wire [4:0]  rt     = InstrIn[20:16];
    wire [4:0]  rd     = InstrIn[15:11];
    wire [5:0]  funct  = InstrIn[5:0];
    wire [15:0] imm    = InstrIn[15:0];
    wire [25:0] addr   = InstrIn[25:0];
    
    wire [31:0] sign_ext_imm = {{16{imm[15]}}, imm};

    //CU
    wire [1:0] RegDst, MemtoReg, ALUOp;
    wire RegWrite, ALUSrc, MemRead, MemWrite, Branch, Jump, Jal, Jr;

    CU control_unit (
        .opcode(opcode),
        .funct(funct),
        .RegDst(RegDst),
        .MemtoReg(MemtoReg),
        .ALUOp(ALUOp),
        .RegWrite(RegWrite),
        .ALUSrc(ALUSrc),
        .MemRead(MemRead),
        .MemWrite(MemWrite),
        .Branch(Branch),
        .Jump(Jump),
        .Jal(Jal),
        .Jr(Jr)
    );
    //

    //RF
    wire [4:0] write_reg;
    wire [31:0] write_data;
    wire [31:0] read_data1, read_data2;

    assign write_reg = (RegDst == 2'b10) ? 5'd31 :
                       (RegDst == 2'b01) ? rt :
                       rd;

    RF register_file (
        .Clk(Clk),
        .Rst(Rst),
        .RegWrite(RegWrite & alu_ready & cpu_ready),
        .ReadReg1(rs),
        .ReadReg2(rt),
        .WriteReg(write_reg),
        .Data(write_data),
        .ReadData1(read_data1),
        .ReadData2(read_data2),
        .R0(R0), .R1(R1), .R2(R2), .R3(R3), .R4(R4), .R5(R5), .R6(R6), .R7(R7),
        .R8(R8), .R9(R9), .R10(R10), .R11(R11), .R12(R12), .R13(R13), .R14(R14), .R15(R15),
        .R16(R16), .R17(R17), .R18(R18), .R19(R19), .R20(R20), .R21(R21), .R22(R22), .R23(R23),
        .R24(R24), .R25(R25), .R26(R26), .R27(R27), .R28(R28), .R29(R29), .R30(R30), .R31(R31)
    );
    //

    //ALU
    wire [31:0] alu_in2;
    wire [31:0] alu_result;
    wire alu_zero;

    assign alu_in2 = ALUSrc ? sign_ext_imm : read_data2;

    ALU alu_inst (
        .clk(Clk),
        .ALUOp(ALUOp),
        .Reg1(read_data1),
        .Reg2(alu_in2),
        .Result(alu_result),
        .zero(alu_zero),
        .ready(alu_ready)
    );
    //
    
    wire valid_MemRead = MemRead & inst_cache_done;
    wire valid_MemWrite = MemWrite & inst_cache_done;
    wire is_memory_instruction = valid_MemRead | valid_MemWrite;
    wire [31:0] safe_data_addr = is_memory_instruction ? alu_result : 32'h00004000;

    L1_Cache data_cache (
        .Clk(Clk),
        .Rst(Rst),
        .CPU_Write(valid_MemWrite),
        .CPU_Read(valid_MemRead),
        .MM_Done(DataMM_Done),
        .CPU_DataIn(read_data2),
        .CPU_Addr(safe_data_addr),
        .MM_DataIn(DataMM_DataIn),
        .CPU_Done(data_cache_done),
        .MM_Write(DataMM_Write),
        .MM_Read(DataMM_Read),
        .CPU_DataOut(DataIn),
        .MM_Addr(DataMM_Addr),
        .MM_DataOut(DataMM_DataOut)
    );
    
    assign cpu_ready = inst_cache_done && (!(valid_MemRead || valid_MemWrite) || data_cache_done);

    wire [31:0] lui_val = {imm, 16'b0};

    assign write_data = (MemtoReg == 2'b11) ? lui_val :
                        (MemtoReg == 2'b10) ? pc_plus_4 :
                        (MemtoReg == 2'b01) ? DataIn :
                        alu_result;

    wire [31:0] branch_target = pc_plus_4 + (sign_ext_imm << 2);
    wire [31:0] jump_target = {pc_plus_4[31:28], addr, 2'b00};
    wire do_branch = Branch & alu_zero;
    
    wire [31:0] pc_after_branch = do_branch ? branch_target : pc_plus_4;
    wire [31:0] pc_after_jump   = Jump ? jump_target : pc_after_branch;
    
    assign next_pc = Jr ? read_data1 : pc_after_jump;

endmodule